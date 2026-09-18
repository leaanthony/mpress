---
title: Dogfennaeth M-Press
description: Adeiladwch ddogfennaeth fodern a chyfoethog gyda Markdown, nid MDX.
layout: landing
translationKey: mpress-home
order: 1
---

@section{variant=hero}
@columns{variant=hero}
@column{variant=hero-copy}
@headline
Dogfennau modern.
Cydrannau cyfoethog.
Dim ond Markdown.
@end

Mae M-Press yn darparu cydrannau cyfoethog, chwilio preifat, rheolyddion hygyrchedd i ddarllenwyr, fersiynau o ryddhadau, a chyfieithu diogel gyda chymorth AI mewn un ffeil ddeuaidd Go gyflym. Dim Node.js. Dim MDX. Dim fframwaith. Mae hyd yn oed y dudalen hon yn Markdown!

@actions
@button[Adeiladu eich gwefan gyntaf](/tutorials/first-site/){primary}
@button[Gweld beth sydd wedi'i gynnwys](/features/){secondary}
@end
@end

@column{variant=site-screenshot|class=mp-home-site-screenshot-hero}
@image{light="/images/mpress-tutorial-light.png" dark="/images/mpress-tutorial-dark.png" alt="The generated M-Press tutorial site" expand}
@end
@end
@end

@section{variant=story|class=mp-home-story-stack}
@columns{variant=story}
@column{variant=story-copy}
## Peidiwch â chynnal pen blaen dim ond er mwyn cyhoeddi geiriau.

Ni ddylai dogfennaeth ddechrau gyda rheolwr pecynnau, uwchraddiad fframwaith, a gwrthdaro mewn ffeil glo. Mae M-Press yn cadw'r model awduro'n fach yn fwriadol: un ffeil ddeuaidd Go, un ffeil YAML, a Markdown cyffredin.

- **Dim amgylchedd rhedeg Node.js.** Gosodwch un ffeil weithredadwy frodorol yn lleol ac mewn CI.
- **Dim dibyniaeth ar MDX.** Mae eich ffynhonnell yn parhau'n ddarllenadwy y tu allan i M-Press.
- **Dim graff adeiladu cudd.** HTML statig cludadwy yw'r canlyniad a gynhyrchir.

@button[Gweld y cyfeirnod nodweddion cyflawn](/features/){secondary}
@end

@column{variant=story-visual|class=mp-home-stack-visual}
@file-tabs
[index.md]

~~~markdown
---
title: Hello world
description: My first M-Press site.
---

# Hello world

This site was built from Markdown.
~~~

[mpress.yaml]

~~~yaml
site:
  title: My Project
  description: My first M-Press site.
build:
  contentDir: docs
  outputDir: site
~~~

[Allbwn]

@image{src="/images/mpress-hello-output.png" alt="A generated Hello world documentation site" expand}
@end
@end
@end
@end

@section{variant=story|class="mp-home-story-components mp-home-story-reversed"}
@columns{variant=story}
@column{variant=story-copy}
## Cydrannau cyfoethog o allweddeiriau Markdown syml.

Mae darllenwyr yn disgwyl tabiau, camau, terfynellau, diagramau, anodiadau cod, coed ffeiliau, a llywio ymatebol. Ni ddylai fod angen MDX na phecyn JavaScript ar awduron ar gyfer pob un. Ysgrifennwch allweddair clir fel `@terminal`, `@steps`, neu `@note`. Mae M-Press yn ei droi'n gydran ryngweithiol, hygyrch.

- **Mae'r chwilio'n awtomatig ac yn breifat.** Mae M-Press yn adeiladu mynegai statig ar gyfer pob iaith. Mae ymholiadau'n aros yn y porwr.
- **Mae un allweddair yn creu'r gydran gyfoethog.** Mae'r ffynhonnell yn parhau'n Markdown byr a darllenadwy.
- **Mae'r cydrannau wedi'u cynnwys.** Nid oes pecyn i'w osod, ei fewnforio na'i ddiweddaru ar gyfer pob patrwm.
- **Mae'r allbwn yn gweithio heb amgylchedd rhedeg fframwaith.** Lletywch ef unrhyw le sy'n gweini ffeiliau statig.

@button[Archwilio pob cydran](/components/){secondary}
@end

@column{variant=story-visual|class=mp-home-components-visual}
@preview-tabs{source}
[Terfynell]
@terminal{title="Build the documentation" language=bash frame=macos}
$ mpress build
Adeiladwyd 60 o dudalennau mewn 47 ms
@end

[Camau]
@steps
### Ysgrifennwch Markdown

Cadwch y ffynhonnell yn ddarllenadwy ym mhob golygydd.

### Adeiladwch y wefan

Cynhyrchwch HTML statig gydag un gorchymyn.
@end

[Nodyn]
@note{type=tip title="Included by default"}
Mae'r gydran, ei thema, ac ymddygiad ei phorwr wedi'u cynnwys gyda M-Press.
@end
@end
@end
@end
@end

@section{variant=story|class="mp-home-story-accessibility mp-home-story-reversed"}
@columns{variant=story}
@column{variant=story-copy}
## Mae hygyrchedd yn rhan o bob gwefan ddogfennaeth.

Ni all un cyflwyniad fod yn addas i bob darllenydd. Mae'r ddewislen hygyrchedd wedi'i galluogi yn ddiofyn ac mae'n storio dewisiadau pob ymwelydd yn ei borwr yn unig. Nid yw perchnogion y wefan yn cael data iechyd na data dewisiadau.

- **Rheolyddion darllen.** Newidiwch faint y testun, lled y dudalen, y ffont, y bylchau, a phwyslais geiriau.
- **Rheolyddion ffocws.** Pylwch bethau sy'n tynnu sylw, dilynwch ganllaw darllen, a lleihau symudiad.
- **Rheolyddion golwg.** Cynyddwch y cyferbyniad, tanlinellwch ddolenni, a dewiswch broffiliau lliw.

@button[Rhoi cynnig ar yr opsiynau hygyrchedd](#){primary|action=accessibility}
@button[Darllen y cyfeirnod hygyrchedd](/accessibility/){secondary}
@end

@column{variant=site-screenshot|class=mp-home-accessibility-screenshot}
@image{light="/images/mpress-accessibility-light.png" dark="/images/mpress-accessibility-dark.png" alt="The open accessibility menu with the complete Reading panel" expand}
@end
@end
@end

@section{variant=story|class=mp-home-story-speed}
@columns{variant=story}
@column{variant=story-copy}
## Daliwch ati i ysgrifennu wrth i'r wefan gadw i fyny.

Mae adeiladu araf yn tarfu ar ganolbwyntio. Mae M-Press yn monitro eich cynnwys, yn ailadeiladu'n awtomatig, ac yn adnewyddu'r porwr. Mae'r un ffeil ddeuaidd hefyd yn gwirio dolenni, yn archwilio ansawdd, yn golygu ffurfweddiad, yn allforio ffeiliau cynhyrchu, ac yn paratoi ar gyfer defnyddio'r wefan.

- **Adborth ar unwaith.** Cadwch ffeil a gwelwch y dudalen a gynhyrchir yn diweddaru.
- **Tystiolaeth weladwy.** Mae pob cam adeiladu a dilysu yn cofnodi ei hyd gwirioneddol.
- **Ansawdd yn yr un llif gwaith.** Rhedwch wiriadau llym ac archwiliad Lighthouse cyn rhyddhau.

@button[Gweld sut mae M-Press yn aros yn gyflym](/explanation/performance/){secondary}
@end

@column{variant=story-visual|class="mp-home-speed-visual mp-home-build-evidence"}
### Mesurwyd ar y set hon o ddogfennau

**60 o dudalennau · 82 o ffeiliau allbwn · 47 i 66 ms**

@capabilities
- **Adeiladu** 47 i 66 ms ar draws dau adeilad cynhyrchu glân
- **Chwilio** Mynegai wedi'i gynhyrchu gyda'r tudalennau
- **Dilysu** Dolenni ac asedau wedi'u casglu wrth rendro
- **Optimeiddio** Lleihau CSS a JavaScript yn frodorol
@end

**Un deuaidd, set gyflawn o offer**

Mae'r gweinydd datblygu, y golygydd ffurfweddu, y gwiriadau, Lighthouse, allforio ZIP, cynorthwywyr defnyddio, mudo, cyfieithu a chipio fersiynau i gyd ar gael o'r un gorchymyn.
@end
@end
@end

@section{variant=story|class=mp-home-story-global}
@columns{variant=story}
@column{variant=story-copy}
## Mae ieithoedd a fersiynau'n rhan o'r craidd.

Mae cyfieithiadau'n peri risg pan gaiff ffensys cod, cystrawen cydrannau a dolenni eu trin fel rhyddiaith arferol. Mae fersiynau'n peri dryswch pan fydd hen dudalennau'n newid heb rybudd. Mae M-Press yn modelu'r ddwy agwedd yn uniongyrchol.

- **Cyfieithu diogel.** Diogelwch y strwythur, defnyddiwch eirfaoedd a chanllawiau arddull, ac olrhain testun hen, testun â llaw, testun sy'n gwrthdaro a thestun sydd wedi'i adolygu.
- **Gwefannau annibynnol ar gyfer pob iaith.** Adeiladwch lwybrau, llywio a chwilio ar gyfer pob iaith.
- **Fersiynau digyfnewid.** Cipiwch, cyfrifwch swm gwirio, dilyswch a gosodwch ryddiadau cyflawn gyda dewisydd ar gyfer darllenwyr.

@button[Darllen am gyfieithu](/translation/){secondary}
@button[Darllen am fersiynau](/versioning/){secondary}
@end

@column{variant=story-visual|class="mp-home-story-image mp-home-global-screenshot"}
@image{light="/images/mpress-language-version-light.png" dark="/images/mpress-language-version-dark.png" alt="The Cymraeg translation of an M-Press documentation site with language and version controls" expand}
@end
@end
@end

@section{variant=story|class="mp-home-story-custom mp-home-story-reversed"}
@columns{variant=story}
@column{variant=story-copy}
## Gweithdy dogfennaeth cyflawn, yn barod yn y modd datblygu.

Mae'r wefan ddatblygu'n dangos y gwaith sydd fel arfer wedi'i guddio ar draws sgriptiau a gwasanaethau. Golygwch bob gosodiad a gefnogir drwy ffurflen strwythuredig, archwiliwch y goeden ffeiliau a gynhyrchir, rhedwch wiriadau, adolygwch berfformiad yr adeilad, dechreuwch archwiliad Lighthouse, allforiwch ZIP a pharatowch y defnydd heb adael y ddogfennaeth.

- **Un ffurfweddiad cyflawn.** Rheolwch hunaniaeth, ieithoedd, dolenni, ffeiliau, thema, hygyrchedd, cynllun, fersiynau, cyfieithu a defnyddio.
- **Yn gwbl addasadwy.** Gosodwch liw, teipograffeg, logos golau a thywyll, llywio, dolenni cymdeithasol a CSS personol.
- **Ymatebol yn ddiofyn.** Yn ddiofyn, mae'r un cynnwys Markdown yn cynnal tudalen lanio lawn ar gyfer cynnyrch a phrofiad darllen symudol â ffocws.

@button[Ffurfweddu gwefan](/configuration/){secondary}
@end

@column{variant=story-visual|class="mp-home-story-image mp-home-config-screenshot"}
@image{light="/images/mpress-config-light.png" dark="/images/mpress-config-dark.png" alt="The structured M-Press configuration editor in development mode" expand}
@end
@end
@end

@section{variant=migration}
@columns{variant=migration}
@column
## Cadwch y cynnwys. Tynnwch y gadwyn offer JavaScript.
@end

@column
Mae'r mewnforiwr yn trosglwyddo tudalennau, cydrannau a gefnogir, llywio, ieithoedd, frontmatter, asedau cyhoeddus a ffurfweddiad. Mae'n ysgrifennu adroddiad ar gyfer pob patrwm y mae angen i berson wneud penderfyniad amdano.

@button[Darllen y canllaw mudo](/starlight/){secondary}
@end
@end
@end

@section{variant=final}
@column
## Adeiladwch ddogfennaeth gyfoethog heb Node.

Crëwch brosiect M-Press gweithredol. Mae hygyrchedd, fersiynau, cyfieithiadau, chwilio, gwiriadau, cydrannau ac optimeiddio cynhyrchu ar gael o'r adeilad cyntaf.
@end

@actions
@button[Dechrau'r tiwtorial](/tutorials/first-site/){primary}
@button[Gweld y ffynhonnell](https://github.com/leaanthony/mpress){secondary}
@end
@end
