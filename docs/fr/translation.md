---
title: Traduire la documentation
description: Créer, mettre à jour et relire le Markdown enrichi M-Press traduit sans modifier le code ni la structure des documents.
translationKey: translation-guide
order: 8
---

M-Press considère la traduction comme partie intégrante du flux de travail de contenu. Il peut
créer du Markdown enrichi M-Press traduit avec un modèle OpenRouter, OpenAI ou compatible OpenAI
de modèle. Le site statique n’a pas besoin du fournisseur ni d’une clé API.

M-Press ne facture pas le flux de travail de traduction, l'acheminement des langues,
le suivi de fraîcheur, les vérifications du glossaire ou l'état de relecture. Le modèle sélectionné
fournisseur peut facturer l'utilisation de l'API.

## Ajouter des langues cibles

Ajoutez chaque langue publiée à `mpress.yaml`:

@explained
```yaml
site: # (1)
  defaultLanguage: en
  languages: [en, fr, ja] # (2)
  languageLabels: # (3)
    en: English
    fr: Français
    ja: 日本語
  defaultLanguageAtRoot: true
  missingTranslation: link-to-default # (4)
```

(1) Placez les paramètres de langue dans la section site du fichier mpress.yaml.

(2) Énumérez les codes de langue que M-Press doit publier. La première build peut n'utiliser
seulement la langue par défaut. Ajoutez un autre code quand son contenu traduit est
prêt.

(3) Définissez les noms visibles par les lecteurs dans le sélecteur de langue.

(4) Redirigez les lecteurs vers la page de la langue par défaut lorsque la traduction sélectionnée
n'existe pas. Définissez cette valeur sur omit pour masquer les langues indisponibles à la place.
@end

La langue par défaut se trouve directement dans le répertoire content. M-Press écrit
les autres langues dans un répertoire de langue :

@explained
```text
content/ # (1)
├── index.md # (2)
├── installation.md
├── _nav.yaml
├── fr/ # (3)
│   ├── index.md
│   ├── installation.md
│   └── _nav.yaml
└── ja/ # (4)
    ├── index.md
    └── _nav.yaml
```

(1) Le répertoire content est l'emplacement par défaut. Modifiez build.contentDir quand
le projet utilise un autre emplacement.

(2) Conservez les pages et la navigation de la langue par défaut directement dans le répertoire content
directory.

(3) Donnez à chaque langue traduite son propre répertoire. Utilisez les mêmes chemins relatifs
que la langue par défaut afin que M-Press puisse apparier les pages.

(4) Un répertoire de langue peut contenir moins de pages. Ici, le japonais n'a pas
installation.md, donc la stratégie de traduction manquante configurée contrôle ce que
le lecteur voit.
@end

Les pages manquantes ne sont pas copiées dans le site généré. `link-to-default` redirige un
visiteur vers la page de la langue par défaut et marque la traduction comme indisponible.
`omit` supprime cette langue du sélecteur de page.

## Configurer un fournisseur

Cet exemple utilise OpenRouter :

```yaml
translation:
  provider: openrouter
  model: openai/gpt-5.4-mini
  baseURL: https://openrouter.ai/api/v1
  apiKeyEnv: OPENROUTER_API_KEY
  sourceLanguage: en
  glossary: glossary.yaml
  styleGuide: docs/writing-standard.md
  stateDir: .mpress/translations
  dataCollection: deny
  requireParameters: true
```

Placez la clé dans l'environnement du processus. Ne la mettez pas dans `mpress.yaml`:

```sh
export OPENROUTER_API_KEY="your-key"
```

Pour OpenAI, utilisez `provider: openai`, `https://api.openai.com/v1`, et
`OPENAI_API_KEY`. Pour un autre fournisseur compatible, utilisez
`provider: openai-compatible` et définissez son URL de base et le nom de la variable d'environnement
name.

Pour la traduction locale en priorité, utilisez un agent de codage installé au lieu d'une
clé API :

```yaml
translation:
  provider: codex # or claude
  model: local
  command: codex # optional; defaults to codex or claude
```

M-Press envoie la requête structurée à la commande via l'entrée standard. La
commande doit renvoyer un objet JSON avec un `translations` tableau contenant le
mêmes `id` valeurs. Cela fonctionne avec une installation locale authentifiée de Codex CLI ou Claude Code
et conserve le contenu de la documentation et les identifiants sur la
machine. Une commande wrapper personnalisée est prise en charge lorsque le CLI local utilise une
invocation différente.

`requireParameters` indique à OpenRouter de sélectionner uniquement un backend qui prend en charge la
réponse structurée. `dataCollection: deny` exclut les fournisseurs qui peuvent stocker
la documentation fournie.

La qualité du modèle varie selon la langue et le jeu de documentation. Dans l'environnement de développement
outil, utilisez le flux de comparaison de modèles pour traduire le même échantillon protégé
avec deux candidats et choisissez le meilleur résultat sans voir leurs noms.
Pour le français et d'autres langues européennes, commencez par comparer
`mistralai/mistral-small-2603`, `openai/gpt-5.4-mini`, et
`google/gemini-3.5-flash-lite`. Traitez-les comme des candidats, pas comme un
classement.

## Vérifier avant de traduire

Exécutez une vérification d'état. Cette commande ne contacte pas le fournisseur :

```sh
mpress translate status --lang fr
mpress translate status --lang fr --file installation.md --json
```

Puis traduisez le texte manquant et obsolète :

```sh
mpress translate --lang fr
mpress translate --lang fr --file installation.md
```

M-Press utilise l'aide de justificatifs Git existante ou
site avec une seule commande:

```sh
mpress translate --lang fr --add-language --label "Français"
```

Utilisez `mpress.yaml`pour combler uniquement les lacunes. Utilisez
toutes les pages Markdown en langue par défaut plus la navigation, et écrit le
résultat sous le répertoire de la langue. L'opération est reprenable. Répéter la
commande traduit uniquement les segments manquants ou périmés.

Vous pouvez lancer le même flux de travail depuis l'
des limites de débit plus élevées, utilisez `--workers` avec une valeur de 1 à 16:

```sh
mpress translate --lang fr --workers 8
```

Utilisez une valeur plus basse lorsque le fournisseur renvoie des erreurs de limite de débit. Chaque page est
validée complètement avant que M-Press remplace son fichier cible et son état.

Si le projet n'est pas extrait sur cette machine, laissez M-Press créer d'abord un
checkout local :

```sh
mpress translate \
  --repo git@github.com:example/docs.git \
  --checkout ../docs-translation \
  --branch main \
  --lang fr
```

M-Press utilise le Git credential helper existant ou `gh` l'authentification. Il
ne copie pas un jeton GitHub dans la configuration du projet. La destination doit être
vide, et le dépôt cloné doit contenir `mpress.yaml`.

Utilisez `--scope missing` pour combler uniquement les lacunes. Utilisez `--scope all --force` seulement quand vous
prévoyez de remplacer le texte généré par machine existant et les conflits manuels.

Vous pouvez exécuter le même flux de travail depuis le **élément Translations** dans la barre de développement
Commencez par le résultat dont vous avez besoin : ajoutez une langue, traduisez la page actuelle
page, mettez à jour une langue complète ou marquez la page actuelle comme relue. M-Press
affiche ensuite la cible, la portée, les pages concernées et la politique de conflits avant
d'écrire le contenu. Les écritures à distance requièrent le jeton d'authoring du serveur de développement.

## Ce que M-Press protège

M-Press analyse le Markdown enrichi M-Press comme un document structuré. Il
envoie des plages de prose exactes au fournisseur et applique le texte renvoyé aux
octets originaux. Il ne régénère pas le document source.

Le traducteur protège :

- le code en bloc et en ligne ;
- les destinations de liens et les liens automatiques ;
- les déclarations HTML et de composants ;
- les directives de composants, les identifiants, les caractères échappés, les références de métadonnées, et
  les shortcodes emoji inconnus ;
- les attributs visibles des composants tels que titres, labels, descriptions, et
  le texte alternatif sont traduits sans exposer leur syntaxe JSON ;
- espaces réservés de configuration tels que `{name}` , et `${HOME}`;
- la ponctuation Markdown et la structure des lignes ;
- les liens de navigation et l'imbrication YAML.

Les requêtes utilisent une sortie structurée indexée par un ID de segment stable. Les pages longues sont
divisées en requêtes limitées avec le titre de la page, le plan, le texte voisin, le guide de style
et le glossaire comme contexte. M-Press rejette les réponses incomplètes et toute
résultat qui modifie la structure protégée.

## Suivre la fraîcheur et les modifications manuelles

La cible correspond à la dernière sortie du fournisseur. `.mpress/translations/` dans le dépôt. Chaque fichier sidecar enregistre
les hachages source et cible, le fournisseur, le modèle, la version du prompt et l'état de relecture. Il
ne contient pas de clé API.

M-Press signale ces états :

| Une personne a modifié la cible pendant que la source restait inchangée. | La source et la cible modifiée manuellement ont toutes deux changé. |
| --- | --- |
| `missing` | ou |
| `machine-translated` | Un outil de relecture a enregistré un état approuvé. |
| `stale` | La source a changé après la traduction automatique. |
| `manual` | Une personne a modifié la cible tandis que la source est restée inchangée. |
| `conflict` | La source et la cible modifiée manuellement ont toutes deux changé. |
| `reviewed` Ajoutez une clé permanente `final` | Un outil de révision a enregistré un état approuvé. |

Les traductions normales préservent le texte manuel et évitent les conflits. Utilisez
`--force` seulement après relecture.

Marquez une page relue depuis l'outil de développement ou le CLI :

```sh
mpress translate review --lang fr --file installation.md
mpress translate review --lang fr --file installation.md --status final
```

Si la source ou la cible change ensuite, M-Press remplace l'approbation par l'état
correct : obsolète, manuel ou conflit.

Ajoutez un permanent `translationKey` au frontmatter quand une page peut se déplacer ou changer
son nom de fichier :

```yaml
---
title: Install M-Press
translationKey: installation
---
```

M-Press utilise cette clé pour déplacer la cible et son état sans renvoyer le texte inchangé
au fournisseur. Les clés doivent être uniques sur tout le site.

## Utiliser un glossaire

Créez un glossaire YAML lorsque le nom d'un produit ou un terme technique doit être homogène :

```yaml
terms:
  - source: M-Press
    translations:
      fr: M-Press
      ja: M-Press
    note: Product name. Do not translate it.
  - source: development server
    translations:
      fr: serveur de développement
```

M-Press inclut les termes applicables dans chaque requête. Il rejette aussi un résultat
lorsqu'une traduction requise est manquante.

## Vérifier les fichiers générés

Les pages traduites sont des fichiers Markdown classiques. Vérifiez et modifiez-les avec les mêmes
outils que les pages sources. Ensuite, exécutez :

```sh
mpress build --strict
mpress check
```

L'outil local fournit la traduction, la gestion d'état, l'application du glossaire,
et la validation. Un produit hébergé peut ajouter une mémoire de traduction partagée, des affectations,
des budgets, des pull requests, des workflows fournisseurs et l'historique d'audit de l'organisation.
