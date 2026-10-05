package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type compressedWriter struct {
	http.ResponseWriter
	writer *gzip.Writer
}

func (w compressedWriter) WriteHeader(status int) {
	w.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(status)
}
func (w compressedWriter) Write(b []byte) (int, error) { return w.writer.Write(b) }
func serveSite(root string) http.Handler {
	files := http.FileServer(http.Dir(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ext := filepath.Ext(r.URL.Path)
		if os.Getenv("MPRESS_LCP_GZIP") == "1" && strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") && (ext == "" || ext == ".html" || ext == ".css" || ext == ".js" || ext == ".svg" || ext == ".json") {
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Add("Vary", "Accept-Encoding")
			z := gzip.NewWriter(w)
			defer z.Close()
			files.ServeHTTP(compressedWriter{w, z}, r)
			return
		}
		files.ServeHTTP(w, r)
	})
}

func measure(url, theme string, width int64) (map[string]any, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath("/usr/bin/google-chrome"), chromedp.NoSandbox)
	alloc, stop := chromedp.NewExecAllocator(context.Background(), opts...)
	defer stop()
	ctx, cancel := chromedp.NewContext(alloc, chromedp.WithLogf(func(string, ...any) {}))
	defer cancel()
	ctx, timeout := context.WithTimeout(ctx, 55*time.Second)
	defer timeout()
	observer := `window.lcpProbe={lcp:[],shifts:[],longTasks:[]};new PerformanceObserver(l=>{for(const e of l.getEntries())window.lcpProbe.lcp.push({time:e.startTime,renderTime:e.renderTime,loadTime:e.loadTime,size:e.size,url:e.url,element:e.element?.outerHTML.slice(0,500)});}).observe({type:'largest-contentful-paint',buffered:true});new PerformanceObserver(l=>{for(const e of l.getEntries())if(!e.hadRecentInput)window.lcpProbe.shifts.push({time:e.startTime,value:e.value});}).observe({type:'layout-shift',buffered:true});new PerformanceObserver(l=>{for(const e of l.getEntries())window.lcpProbe.longTasks.push({time:e.startTime,duration:e.duration});}).observe({type:'longtask',buffered:true});try{localStorage.setItem('mpress-theme',` + strconv.Quote(theme) + `);localStorage.setItem('theme',` + strconv.Quote(theme) + `);}catch{}`
	err := chromedp.Run(ctx, chromedp.EmulateViewport(width, 844), emulation.SetCPUThrottlingRate(4), network.Enable(), network.ClearBrowserCache(), network.SetCacheDisabled(true), chromedp.ActionFunc(func(ctx context.Context) error {
		_, err := network.EmulateNetworkConditionsByRule([]*network.Conditions{{URLPattern: "", Latency: 150, DownloadThroughput: 200 * 1024, UploadThroughput: 100 * 1024}}).Do(ctx)
		return err
	}), chromedp.ActionFunc(func(ctx context.Context) error {
		_, err := page.AddScriptToEvaluateOnNewDocument(observer).Do(ctx)
		return err
	}), chromedp.Navigate(url), chromedp.Sleep(4*time.Second))
	if err != nil {
		return nil, err
	}
	var r map[string]any
	err = chromedp.Run(ctx, chromedp.Evaluate(`({...window.lcpProbe,url:location.href,title:document.title,theme:document.documentElement.dataset.theme,navigation:performance.getEntriesByType('navigation').map(e=>e.toJSON()),paint:performance.getEntriesByType('paint').map(e=>e.toJSON()),resources:performance.getEntriesByType('resource').map(e=>e.toJSON()),images:Array.from(document.images,e=>({src:e.src,currentSrc:e.currentSrc,loading:e.loading,priority:e.fetchPriority,decoding:e.decoding,complete:e.complete,w:e.naturalWidth,h:e.naturalHeight,display:getComputedStyle(e).display,rect:e.getBoundingClientRect().toJSON()}))})`, &r))
	r["viewportWidth"] = width
	r["cpuSlowdown"] = 4
	r["latencyMs"] = 150
	r["downloadBytesPerSecond"] = 200 * 1024
	return r, err
}

func main() {
	target := os.Args[1]
	if !strings.HasPrefix(target, "http") {
		server := httptest.NewServer(serveSite(target))
		defer server.Close()
		target = server.URL + "/"
	}
	if route := os.Getenv("MPRESS_LCP_ROUTE"); route != "" {
		target = strings.TrimRight(target, "/") + "/" + strings.TrimLeft(route, "/")
	}
	runs := 1
	if len(os.Args) > 2 {
		runs, _ = strconv.Atoi(os.Args[2])
	}
	var records []map[string]any
	for _, width := range []int64{1365, 390} {
		for _, theme := range []string{"light", "dark"} {
			for n := 0; n < runs; n++ {
				r, err := measure(target, theme, width)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				r["run"] = n + 1
				r["gzip"] = os.Getenv("MPRESS_LCP_GZIP") == "1"
				records = append(records, r)
			}
		}
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))
}
