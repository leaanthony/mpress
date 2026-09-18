---
title: Créez votre premier site M-Press
description: Installez M-Press, créez un site, modifiez une page et vérifiez le résultat.
order: 2
---

Ce tutoriel vous donne un site de documentation fonctionnel. Vous devez avoir Go 1.26.5 ou une version ultérieure.

## Installer M-Press

Exécutez cette commande :

```sh
go install github.com/leaanthony/mpress/cmd/mpress@latest
```

Vérifiez que l'exécutable est disponible :

```sh
mpress version
```

## Créer le site

Exécutez ces commandes :

```sh
mpress init product-docs
cd product-docs
mpress dev
```

M-Press affiche l'adresse locale. Ouvrez cette adresse dans votre navigateur.

Vous avez maintenant ce projet :

@filetree
product-docs/
  mpress.yaml  Configuration du site
  content/
    index.md  Page d'accueil
  static/  Images et autres fichiers statiques
@end

## Modifier la page d'accueil

Ouvrez `content/index.md`. Les deux fichiers source produisent la page affichée
dans le troisième onglet :

@tabs
[content/index.md]
```md {title="content/index.md"}
---
title: Product documentation
description: Learn how to use the product.
---

Welcome to the product documentation.

## Install the product

Follow the installation steps for your operating system.
```

[mpress.yaml]
```yaml {title="mpress.yaml"}
site:
  title: Product documentation
  description: Learn how to use the product.
build:
  contentDir: content
  staticDir: static
  outputDir: site
search:
  enabled: true
accessibility:
  enabled: true
```

[Résultat]
@image{light="/images/mpress-tutorial-output-light.png" dark="/images/mpress-tutorial-output-dark.png" alt="Le site de documentation généré dans M-Press" expand}
@end

Enregistrez le fichier. Le serveur de développement reconstruit le site et actualise la page.

## Vérifier la publication

Arrêtez le serveur de développement. Ensuite, exécutez ces commandes :

```sh
mpress build --strict
mpress check
```

Le `site/` répertoire contient maintenant le site statique. La compilation stricte signale
le contenu non pris en charge et les routes dupliquées comme erreurs. La vérification signale des
liens locaux, fragments, scripts, feuilles de style et images.

Vous avez construit et vérifié votre premier site M-Press.

## Continuer

- [Créer une page d'accueil personnalisée](/how-to/custom-landing-page/).
- [Définir la navigation](/navigation/).
- [Lire la référence de création de contenu](/authoring/).
