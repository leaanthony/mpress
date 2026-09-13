---
title: Référence de la ligne de commande
description: Commandes pour créer, développer, valider, importer et versionner des sites M-Press.
order: 3
---

Exécutez `mpress help` pour afficher le résumé compact des commandes.

## Commandes du projet

| Commande | Objet |
| --- | --- |
| `mpress init [directory]` | Créez un projet de démarrage. Le répertoire par défaut est le répertoire courant. |
| `mpress dev [directory] [--port 3000]` | Générez des brouillons, servez la sortie, surveillez les fichiers source et rechargez en direct. |
| `mpress dev --repo URL --checkout DIRECTORY [--branch NAME]` | Clonez un dépôt, créez une branche de travail sécurisée et lancez son interface de développement. |
| `mpress contribute SITE_URL_OR_REPOSITORY [--branch BRANCH] [--checkout DIRECTORY] [--draft-file FILE] [--no-open]` | Découvrez une page ou utilisez un dépôt configuré, appliquez un brouillon de navigateur téléchargé en option, préparez un checkout local et ouvrez l'assistant de contribution. |
| `mpress build [--strict] [--drafts] [--json] [--no-purge-css]` | Générez le site de production avec CSS et JavaScript minifiés. |
| `mpress export [archive.zip] [--strict] [--drafts] [--force] [--json]` | Générez et empaquetez le site statique complet en archive ZIP. |
| `mpress clean` | Supprimez le répertoire de sortie configuré. |
| `mpress check` | Vérifiez les liens locaux générés, les fragments et les ressources. |
| `mpress translate status [--lang CODE] [--file PAGE] [--json]` | Signalez les traductions manquantes, obsolètes, manuelles et en conflit sans utiliser une API. |
| `mpress translate [--lang CODE] [--file PAGE] [--scope missing\|stale\|all] [--force]` | Traduisez Markdown et la navigation avec le fournisseur configuré. |
| `mpress translate review --lang CODE --file PAGE [--status reviewed\|final]` | Enregistrez l'état de relecture d'une page traduite. |
| `mpress translate --repo URL --checkout DIRECTORY [--branch NAME] ...` | Clonez un checkout local manquant, puis exécutez la traduction dans celui-ci. |
| `mpress version` | Affichez la version de M-Press installée. |

M-Press recherche le répertoire courant et ses parents `mpress.yaml`, ainsi
les commandes peuvent être exécutées depuis un répertoire de contenu imbriqué.

Démarrez un projet existant sans changer votre répertoire shell :

```sh
mpress dev ../product-docs
```

Pour commencer à partir d'un référentiel distant, utilisez une commande. L'authentification Git provient
de votre credential helper existant ou `gh` login. M-Press clone la source,
crée une branche de travail non par défaut, génère le site et affiche l'URL de l'interface locale :

```sh
mpress dev \
  --repo git@github.com:example/product-docs.git \
  --checkout ../product-docs \
  --branch docs/add-french
```

Ouvrez l'URL affichée et utilisez **Translations** dans la barre de développement. La
les outils d'édition reconnaissent la branche préparée, donc ils ne vous demandent pas de préparer
le référentiel une seconde fois.

Démarrez à partir d'une page M-Press publiée lorsque le site a activé les contributions :

```sh
mpress contribute https://docs.example.com/guide/install/
```

M-Press découvre le référentiel et le fichier source correspondant à partir de la page. Il
utilise `~/mpress-contributions/<owner>-<repository>` par défaut, crée une
branche de contribution sécurisée, sélectionne un port disponible et ouvre l'assistant local. Utilisez
`--checkout` pour choisir un autre répertoire ou `--no-open` pour garder le navigateur
fermé.

Quand l'édition rapide télécharge un brouillon du navigateur, passez son nom de fichier avec
`--draft-file`. M-Press recherche le répertoire courant et votre répertoire Downloads configuré.
Vous pouvez aussi fournir un chemin complet.

## Modes de génération

@tabs
[Normal]
`mpress build` émet des diagnostics mais termine la génération. Le contenu non pris en charge est
rendu visible dans la sortie au lieu de disparaître.

[Strict]
`mpress build --strict` écrit la sortie et renvoie un échec lorsque des diagnostics de niveau erreur
existent. Utilisez ceci en CI et avant chaque publication.

[Lisible par machine]
`mpress build --json` affiche le nombre de pages, le nombre de fichiers, la durée de génération et
les diagnostics au format JSON. Ceci est utile pour les intégrations et les contrôles de publication privés.
@end

Les pages brouillon sont exclues sauf si `--drafts` est fourni. Le serveur de développement
les inclut automatiquement.

Les builds de production minifient le CSS et le JavaScript générés avec le minificateur Go intégré de M-Press.
Il supprime les commentaires et les espaces blancs redondants et raccourcit
les liaisons JavaScript locales sûres non répétées. Il n'a pas besoin de Node ni d'un outil
de minification externe. Les builds de production suppriment aussi les règles de classe et d'ID clairement inutilisées
après avoir analysé chaque page générée et le script d'exécution. Les at-rules responsive
sont filtrées de façon récursive, tandis que les at-rules d'animation, de police et inconnues
sont conservées. Les sélecteurs sans jetons de classe ou d'ID sont conservés car des composants
HTML et les composants d'exécution peuvent les activer plus tard. Passez `--no-purge-css` lorsque
une génération doit conserver chaque sélecteur.

## Exportations ZIP

Exécutez `mpress export` pour créer `<project-directory>.zip` dans la racine du projet. L'
archive contient le HTML de production, le CSS, le JavaScript, les index de recherche et
les ressources statiques à sa racine, prêts pour tout hébergeur statique. Elle ne contient pas les sources
Markdown, la barre de développement ou les points de terminaison d'authoring.

Utilisez une destination explicite lorsque un autre outil attend un nom d'artefact fixe :

```sh
mpress export --strict docs-release.zip
```

M-Press refuse de remplacer une archive existante sauf si `--force` est fourni.
Utilisez `--drafts` uniquement pour les archives de revue privées.

## Migration

```sh
mpress import --from starlight ../old-docs --output ./new-docs
```

Les options peuvent apparaître avant ou après le chemin source. L'importateur écrit un
`migration-report.md` à côté de la nouvelle configuration du projet.

## Artefacts de version

```sh
mpress versions capture v1.0
mpress versions capture --force v1.0
mpress versions list
mpress versions verify v1.0
mpress versions remove v1.0
```

Les étiquettes de version peuvent contenir des lettres, des chiffres, des points, des traits de soulignement et des tirets.

## Traduction

`mpress translate` lit sa clé API depuis la variable d'environnement nommée par
`translation.apiKeyEnv`. Il n'écrit jamais la clé dans les fichiers du projet. Les exécutions normales
traduisent les segments manquants et obsolètes et préservent les modifications manuelles. Voir
[Translate documentation](/translation/) pour les détails du fournisseur, du glossaire, de l'état et
les détails de la revue.
