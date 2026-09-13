---
title: Markdown et HTML
description: Structurez le contenu M-Press avec Markdown, HTML, frontmatter, liens, ressources et brouillons.
order: 5
---

Les fichiers de contenu utilisent `.md` ou `.markdown`. M-Press prend en charge CommonMark plus les tableaux,
les listes de tâches, le texte barré, les notes de bas de page, les substitutions typographiques et les
identifiants de titre.

## Flux de développement

Exécutez `mpress dev` et ouvrez le site généré. Une barre réservée au développement apparaît en
bas de chaque page. Utilisez son menu M-Press pour modifier la configuration courante,
exécuter les vérifications de publication, créer un déploiement ou redémarrer le guide du projet.

La première action d'édition ouvre **Setup** lorsque le projet en cours n'est pas sur une
branche de travail. Continuez sur le checkout actuel, clonez un dépôt, ou créez et
clonez un fork GitHub. M-Press crée une branche de travail avant d'ouvrir les outils d'édition.
outils.

Pour le flux le plus court avec un dépôt existant, clonez-le et démarrez-le avec une seule commande CLI
commande:

```sh
mpress dev --repo git@github.com:example/docs.git \
  --checkout ../docs-work \
  --branch docs/improve-site
```

L'interface de développement ouvre directement les outils d'édition parce que le checkout est déjà
sur la branche préparée. Si vous clonez ou forkez depuis **Setup** à la place, copiez la
commande `mpress dev "/path/to/checkout"` montrée lorsque la préparation se termine.

Un site publié peut offrir un flux lecteur encore plus court. L'action **Contribute**
donne au lecteur une commande unique qui télécharge M-Press et exécute
`mpress contribute <page-url>`. M-Press trouve le dépôt et le fichier source,
prépare une branche locale, ouvre la page correspondante et demande ce que le lecteur souhaite
améliorer. Voir [Contribuer depuis une page publiée](/how-to/enable-site-contributions/).

La configuration s'ouvre dans un panneau vitré déplaçable sur bureau et dans une feuille contenue
sur mobile. La page reste visible et défilable derrière. Le titre du site, le schéma de couleur,
la couleur d'accent et la couleur au survol se mettent à jour lorsque vous modifiez leurs champs.
La fermeture du panneau restaure les valeurs non enregistrées. L'enregistrement valide `mpress.yaml`,
reconstruit le site et conserve les nouvelles valeurs.

Éditez le Markdown dans votre éditeur de texte habituel. M-Press surveille les fichiers du projet, reconstruit
le site et recharge le navigateur après un changement réussi. La barre de développement
et les endpoints d'écriture ne sont pas inclus dans `mpress build` la sortie.

## Tester une page avec Lighthouse

Ouvrez le menu M-Press et sélectionnez **Lighthouse audit**. M-Press teste la page
ouverte dans le navigateur et affiche des scores pour les performances, l'accessibilité,
les bonnes pratiques et le SEO. Vous pouvez utiliser le profil de test mobile ou desktop.

L'audit est optionnel. La construction et le service d'un site M-Press n'exigent pas
Node.js. L'audit utilise une commande `lighthouse` commande lorsqu'une est
disponible. Sinon, il peut exécuter Lighthouse via `npx` après que vous ayez explicitement
démarré l'audit. Les versions actuelles de Lighthouse requièrent Node.js 22 ou ultérieur et une
installation locale de Chrome ou Chromium.

Définissez `MPRESS_LIGHTHOUSE` sur le chemin ou le nom de commande d'un exécutable Lighthouse
lorsqu'il est installé dans un emplacement non standard.

## Connecter un agent à MCP

`mpress dev` inclut un serveur MCP dans le même binaire. Le terminal imprime le
endpoint et un jeton de serveur aléatoire lorsque le développement démarre:

```text
M-Press MCP:    http://localhost:3000/__mpress/mcp
MCP token:      4c21...
```

Configurez un client MCP pour utiliser Streamable HTTP et envoyer le jeton comme credential Bearer
credential:

```json
{
  "mcpServers": {
    "mpress": {
      "url": "http://localhost:3000/__mpress/mcp",
      "headers": {
        "Authorization": "Bearer 4c21..."
      }
    }
  }
}
```

Le serveur MCP peut inspecter le projet, lister et éditer les fichiers source, mettre à jour la
configuration complète, exécuter des vérifications, capturer des versions et déployer les cibles configurées.
cibles. Les écritures de fichiers et de configuration requièrent la révision renvoyée par l'
outil de lecture correspondant. M-Press crée une sauvegarde et reconstruit le site après un
changement réussi.

L'endpoint MCP n'accepte pas le token dans son URL. Envoyez le même token sur
chaque requête en utilisant l'en-tête `Authorization` header. Le redémarrage de `mpress dev` crée
un nouveau token sauf si vous fournissez un token stable avec `mpress dev --token VALUE`.

## Frontmatter

```yaml
---
title: Build an application
description: Create and package your first application.
slug: guides/first-application
order: 20
draft: false
layout: landing
tags: [guide, beginner]
author: Documentation team
---
```

`title` est recommandé. Sans titre, M-Press utilise la première rubrique, puis
le nom de fichier. `slug` remplace la route. `order` contrôle l'ordonnancement de la
navigation générée. Les brouillons apparaissent dans `mpress dev` et `mpress build --drafts`. Ils ne
apparaissent dans une build de publication normale.

La mise en page par défaut est un article de documentation. Définissez `layout: landing` pour remplacer
les colonnes de documentation par une mise en page Markdown écrite. L'en-tête commun
reste. M-Press n'ajoute pas de rubrique automatique ni de liens de page à une page d'accueil
de destination. Voir [Créer une page d'accueil Markdown](/how-to/custom-landing-page/).

## Routes

| Source | Route |
| --- | --- |
| `index.md` | `/` |
| `installation.md` | `/installation/` |
| `guides/index.md` | `/guides/` |
| `01-guides/02-build.md` | `/guides/build/` |

Les préfixes numériques aident à ordonner les fichiers sans entrer dans l'URL publique.
`guide.md` et `guide/index.md` entrent donc en collision ; les builds stricts signalent le
problème plutôt que d'en choisir un silencieusement.

## Liens et ressources

Les liens de documentation relatifs à la racine sont généralement les plus clairs :

```md
[Install M-Press](/getting-started/)
![Application window](/images/application.png)
```

Placez l'image à `static/images/application.png`. Exécutez `mpress check` après
la construction pour détecter les cibles et fragments manquants.

## Images claires et sombres

Utilisez le composant image natif quand une image ne fonctionne pas dans les deux modes de couleur
modes :

```md
@image{light="/images/architecture-light.png" dark="/images/architecture-dark.png" alt="System architecture"}
```

Les deux fichiers restent des ressources statiques ordinaires. M-Press affiche l'image qui correspond
au mode de couleur choisi par le visiteur. Fournissez toujours un texte alternatif utile.

Ajoutez l'option `expand` lorsque les lecteurs doivent examiner une version plus
grande. Un clic sur l'image l'ouvre dans une superposition centrée. Le bouton de
fermeture, l'arrière-plan ou la touche Échap ferment la superposition.

```md
@image{src="/images/architecture.png" alt="Architecture du système" expand}
```

## HTML ordinaire

Le HTML passe par le moteur de rendu lorsque le Markdown n'est pas suffisant :

```html
<details>
  <summary>Show advanced details</summary>
  This remains ordinary, portable HTML.
</details>
```

M-Press n'exécute pas de JSX ni de composants de framework. Un composant MDX inconnu en majuscules
tel que `&lt;ProductDemo /&gt;` devient un marqueur d'avertissement visible et un
diagnostic d'erreur. Cela évite que le contenu de migration ne disparaisse sans être remarqué.
