package site

const svgObjectSizingJS = `
  const svgObjectRatios = new WeakMap();
  const resizeSVGObject = object => {
    const ratio = getComputedStyle(object).aspectRatio;
    if (object.hasAttribute('height') || object.style.height || (ratio !== 'auto' && ratio !== svgObjectRatios.get(object))) return;
    let svg;
    try { svg = object.contentDocument?.documentElement; } catch (_) { return; }
    if (svg?.localName !== 'svg') return;
    const box = svg.viewBox?.baseVal;
    const lengths = [svg.width?.baseVal, svg.height?.baseVal];
    const absolute = lengths.every(length => length && (length.unitType === 1 || length.unitType >= 5));
    const width = absolute ? lengths[0].value : box?.width;
    const height = absolute ? lengths[1].value : box?.height;
    if (!(width > 0 && height > 0 && Number.isFinite(width) && Number.isFinite(height))) return;
    object.style.aspectRatio = width + ' / ' + height;
    svgObjectRatios.set(object, getComputedStyle(object).aspectRatio);
  };
  document.querySelectorAll('article object[type="image/svg+xml"]').forEach(object => {
    object.addEventListener('load', () => resizeSVGObject(object));
    resizeSVGObject(object);
  });
`
