---
title: Installation et votre premier site
description: Installez M-Press et générez un site de documentation en quelques commandes.
order: 2
---

M-Press nécessite uniquement `mpress` l'exécutable au moment de l'exécution.

## Compiler depuis les sources

Avant la publication des binaires de version empaquetés, installez depuis le module Go :

```sh
go install github.com/leaanthony/mpress/cmd/mpress@latest
```

Vous pouvez aussi construire un checkout du dépôt :

```sh
git clone https://github.com/leaanthony/mpress.git
cd mpress
go build -o mpress ./cmd/mpress
```

@note{type="info" title="Release packaging"}
Des binaires précompilés pour Linux, macOS et Windows font partie du plan de publication 0.1.
La compilation depuis les sources est le chemin d'installation actuel.
@end

## Créer un site

```sh
mpress init my-docs
cd my-docs
mpress dev
```

Ouvrez `http://localhost:3000`. Le starter contient :

@filetree
my-docs/
  mpress.yaml  Configuration du site
  content/
    index.md  Page d'accueil
  static/  Images et autres fichiers statiques
@end

Éditez `content/index.md` et enregistrez-le. Le serveur de développement reconstruit le site
et rafraîchit les onglets de navigateur connectés.

## Générer pour la publication

```sh
mpress build --strict
mpress check
```

`--strict` rejette les routes en double, les composants non pris en charge et d'autres
diagnostics. `mpress check` vérifie les liens locaux, les fragments, les images, les scripts,
et les feuilles de style dans le HTML généré.

Le répertoire de sortie par défaut est `site/`. Il peut être servi directement :

```sh
python3 -m http.server --directory site 8000
```

## Utiliser M-Press dans un checkout Go existant

M-Press utilise lui-même cette approche. Placez `mpress.yaml` à la racine du dépôt,
définissez `build.contentDir` sur votre répertoire de documentation, puis exécutez :

```sh
go run ./cmd/mpress build --strict
go run ./cmd/mpress check
```
