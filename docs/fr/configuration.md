---
title: Configuration
description: Configurer les métadonnées du site, les chemins de build, la recherche, le thème, l'accessibilité, la contribution, les langues, les liens et les versions.
order: 4
---

Chaque projet a un `mpress.yaml`. Les champs inconnus sont rejetés, donc les fautes d'orthographe
ne puissent pas modifier silencieusement une publication.

```yaml
site:
  title: Acme Documentation
  description: Learn how to build with Acme.
  baseURL: https://docs.example.com
  defaultLanguage: en
  languages: [en, fr]
  languageLabels:
    en: English
    fr: Français
  defaultLanguageAtRoot: true
  missingTranslation: link-to-default
  logoLight: logo-light.svg
  logoDark: logo-dark.svg
  logoWidth: 160px
  favicon: favicon.svg
  socialImage: social-card.png
  headerLinks:
    - label: Guides
      url: /guides/
    - label: API
      url: /reference/

social:
  github: https://github.com/example/acme
  discord: https://discord.gg/example
  x: https://x.com/example
  editURL: https://github.com/example/acme/edit/main/docs

contribution:
  enabled: true
  repository: https://github.com/example/acme.git
  branch: main

theme:
  accentColor: "#5375f6"
  hoverColorLight: "#315bd6"
  hoverColorDark: "#ffffff"
  colorScheme: system
  layout:
    preset: starlight

build:
  contentDir: content
  staticDir: static
  outputDir: site
  navFile: _nav.yaml
  customCSS: custom.css

blog:
  showTags: true
  headingSize: default
  imageMode: panel
  imageFit: cover
  imageWidth: 100
  imageBackground: ""

search:
  enabled: true
  shortcut: Mod+K

accessibility:
  enabled: true
  shortcut: Mod+A

versioning:
  enabled: false
  current: next
  artifactsDir: .mpress/versions

translation:
  provider: openrouter
  model: openai/gpt-5-mini
  baseURL: https://openrouter.ai/api/v1
  apiKeyEnv: OPENROUTER_API_KEY
  sourceLanguage: en
  glossary: glossary.yaml
  styleGuide: docs/writing-standard.md
  stateDir: .mpress/translations
  dataCollection: deny
  requireParameters: true
```

## Site

`title` est requis. `description` devient la description de page de secours.
`baseURL` active les URL canoniques, `sitemap.xml`, l'entrée du plan du site dans
`robots.txt`, et les métadonnées absolues de partage sur les réseaux sociaux. Laissez-le vide pour
les builds locaux.

Les logos, le favicon et les chemins référencés depuis Markdown doivent exister sous le
répertoire statique configuré. Les fichiers statiques sont copiés dans la racine de sortie.

Utilisez `logoLight` pour un logo qui a un contraste suffisant en mode clair. Utilisez
`logoDark` pour le logo en mode sombre. M-Press change le logo lorsque le visiteur
change le mode de couleur. `logoWidth` contrôle la largeur rendue de l'un ou l'autre
logo du thème et accepte une valeur positive `px`, `rem`, ou `ch` valeur. En mode dev, la
page Paramètres de marque peut téléverser des fichiers SVG, PNG, JPEG, WebP ou AVIF directement vers
du projet `static/brand` répertoire.

`socialImage` est optionnel. Utilisez une image proche de 1200 par 630 pixels pour les aperçus enrichis
Open Graph et Twitter. M-Press écrit toujours les titres, les descriptions,
et les types de page. Il ajoute les métadonnées d'image lorsque les deux `baseURL` et
`socialImage` sont définis.

M-Press génère un `404.html` et un `robots.txt`. Ajoutez l'un ou l'autre
des fichiers dans le répertoire statique lorsque le projet a besoin d'une version personnalisée.

`headerLinks` ajoute de courts liens textuels à côté de l'identité du site. Gardez cette liste
petite. Les contrôles de recherche, de version, de langue, sociaux et de thème restent dans le
groupe de contrôle à droite.

## Contribution

Définissez `contribution.enabled` pour ajouter une **Contribuer** action aux pages générées.
`repository` est le dépôt Git qui contient `mpress.yaml`. `branch` est la
branche source à cloner et par défaut `main`.

La page générée publie les métadonnées du dépôt et des fichiers source. Elle ne
publie jamais les identifiants. Les dépôts privés utilisent l'authentification Git existante du lecteur.
l'authentification. Voir [Contribuer depuis une page publiée](/how-to/enable-site-contributions/) pour le
flux de travail du lecteur et les règles de sécurité du checkout.

## Blog

La page des paramètres du Blog contrôle la présentation par défaut de l'archive de blog générée
d'archive. Les articles peuvent remplacer ces valeurs dans le frontmatter. `showTags` contrôle
la rangée d'étiquettes, `headingSize` peut être `default`, `compact`, ou `large`, et
`imageMode` peut être `panel` ou `floating`. `imageFit` peut être `cover` ou
`contain`. `imageWidth` contrôle les images flottantes de 50 à 100 pour cent.
`imageBackground` accepte une couleur hexadécimale à six chiffres et est ignoré pour les images flottantes
images.

## Thème

`colorScheme` peut être `system`, `light`, ou `dark`. Les visiteurs peuvent le changer via
le contrôle du thème ; leur choix est stocké localement dans le navigateur.

`accentColor` change la couleur interactive principale. `hoverColorLight` et
`hoverColorDark` définissent la couleur utilisée lorsque le pointeur survole les icônes de navigation
dans chaque thème. L'ancien `hoverColor` champ reste un repli pour les projets existants
projets. Pour une personnalisation plus poussée, définissez `build.customCSS` sur une feuille de style
relative à la racine du projet. Elle est copiée après la feuille de style par défaut afin que vos
règles peuvent remplacer le thème.

Le `starlight` préréglage de mise en page est le défaut. Il fixe la navigation à la
gauche et centre l'article et la table des matières comme une seule unité. Le `wide`
préréglage donne plus d'espace aux pages de référence. Le `reading` préréglage utilise une
longueur de ligne pour la prose.

Utilisez `custom` lorsque le projet a besoin de mesures exactes :

```yaml
theme:
  layout:
    preset: custom
    contentWidth: 720px
    wideContentWidth: 1024px
    sidebarWidth: 300px
    tocWidth: 256px
    contentTocGap: 40px
    alignment: cluster
    toc: right
```

Les largeurs d'article acceptent `px`, `rem`, `ch`, ou un pourcentage de `40%` à `100%`.
Les autres mesures acceptent `px`, `rem`, ou `ch`. Utilisez `alignment: left` pour garder
l'unité de contenu près de la navigation. Utilisez `toc: hidden` pour supprimer la
table des matières de bureau. La mise en page se replie toujours pour les écrans plus petits.

Ajoutez `layout: wide` au frontmatter de la page lorsqu'une page de tableau ou de référence nécessite la
largeur de page large configurée.

## Accessibilité

Le menu d'accessibilité est activé par défaut. Il permet à chaque visiteur de modifier la taille du texte
et l'espacement, d'utiliser une police lisible ou la lecture bionique, de réduire les distractions
et le mouvement, d'ajouter un guide de lecture, d'augmenter le contraste, de souligner les liens, et de choisir
un profil de couleur.

Les préférences restent dans le navigateur du visiteur. M-Press ne les envoie pas à un
serveur. Définissez `accessibility.enabled: false` pour supprimer le contrôle, le panneau,
les règles de feuille de style et le script du site généré.

## Raccourcis clavier

`search.shortcut` ouvre la recherche. `accessibility.shortcut` ouvre le menu d'
accessibilité. `Mod` signifie Command sur les appareils Apple et Ctrl sur les autres plateformes. Chaque
raccourci doit contenir au moins un modificateur et une touche. Définissez un raccourci sur
`None` pour le désactiver. M-Press n'intercepte pas les raccourcis pendant que le lecteur est
en train d'éditer un input, un textarea, un select, ou une zone éditable.

## Build paths and safety

Les chemins de build relatifs sont résolus à partir du projet contenant `mpress.yaml`.
Le répertoire de sortie doit être un enfant de ce projet. M-Press refuse de nettoyer
ou de remplacer la racine du projet, son parent, ou un répertoire externe.

## Recherche

La recherche statique produit un petit index JSON par langue. La recherche s'exécute entièrement
dans le navigateur et n'envoie aucune requête à un service.

## Traduction

Les paramètres de traduction s'appliquent uniquement lorsque vous exécutez `mpress translate` ou utilisez l'
outil de développement Translations. Les builds statiques ne contactent pas de fournisseur.

`apiKeyEnv` nomme une variable d'environnement. Ne mettez pas les identifiants eux-mêmes dans
la configuration. `stateDir` stocke les métadonnées de fraîcheur et de revue engagées.
`glossary` , et `styleGuide` sont des chemins optionnels à l'intérieur du projet. Voir
[Traduire la documentation](/translation/) pour le flux de travail complet.
