---
title: Composants interactifs
description: Amélioration progressive sans framework JavaScript.
order: 9
---

Les composants interactifs sont d'abord du HTML statique. Le thème par défaut ajoute un petit,
script sans dépendances pour l'état, les requêtes réseau et la présentation.

## Tutoriels

@tutorial{title="Publish a site"}
### Vérifiez le contenu
Exécutez `mpress check`.

### Construisez strictement
Exécutez `mpress build --strict`.

### Déployez la sortie
Téléversez le `site/` répertoire généré.
@end

@details{title="Source"}
```md
@tutorial{title="Publish a site"}
### Vérifiez le contenu
Exécutez `mpress check`.

### Construisez strictement
Exécutez `mpress build --strict`.

### Déployez la sortie
Téléversez le `site/` répertoire généré.
@end
```
@end

La progression est stockée localement et le bouton de réinitialisation la supprime.

## Filtrage d'audience

Le sélecteur est ajouté automatiquement lorsqu'une page contient des blocs d'audience.

@audience{role="developer"}
Les développeurs peuvent étendre le pipeline de traitement Markdown en Go.
@end

@audience{role="writer"}
Les rédacteurs n'ont besoin que de Markdown, HTML et de la commande preview.
@end

@details{title="Source"}
```md
@audience{role="developer"}
Les développeurs peuvent étendre le pipeline de traitement Markdown en Go.
@end

@audience{role="writer"}
Les rédacteurs n'ont besoin que de Markdown, HTML et de la commande preview.
@end
```
@end

## Contenu conditionnel

Les blocs conditionnels lisent un paramètre d'URL, puis le mémorisent localement. Visitez cette
page avec `?framework=go` pour sélectionner le deuxième bloc.

@if{param="framework" value="plain" default="true"}
Ceci est la consigne par défaut, indépendante du framework.
@end

@if{param="framework" value="go"}
Cette consigne est sélectionnée pour les utilisateurs de Go.
@end

@details{title="Source"}
```md
@if{param="framework" value="plain" default="true"}
Ceci est la consigne par défaut, indépendante du framework.
@end

@if{param="framework" value="go"}
Cette consigne est sélectionnée pour les utilisateurs de Go.
@end
```
@end

## Valeurs réactives

@input{name="projects" type="range" min="1" max="20" value="4" label="Projects"}

@input{name="seats" type="number" value="3" label="Editors"}

@computed{expr="projects * seats" deps="projects,seats" label="Project seats" format="%d"}

@details{title="Source"}
```md
@input{name="projects" type="range" min="1" max="20" value="4" label="Projects"}

@input{name="seats" type="number" value="3" label="Editors"}

@computed{expr="projects * seats" deps="projects,seats" label="Project seats" format="%d"}
```
@end

## Bac à sable API

Le bac à sable n'envoie une requête réelle depuis le navigateur que lorsque le lecteur appuie sur le
bouton. Cet exemple cible le serveur de preview local et est sans danger à inspecter.

@api-playground{method="GET" path="/search-index.json" baseUrl="http://127.0.0.1:4174"}
Inspectez l'index de recherche généré.
@end

@details{title="Source"}
```md
@api-playground{method="GET" path="/search-index.json" baseUrl="http://127.0.0.1:4174"}
Inspectez l'index de recherche généré.
@end
```
@end

## Variantes

`@variant{name="react"}` est résolu lors du build plutôt que dans le
navigateur. Définissez la variante active dans la configuration de build pour publier un jeu
de contenu spécifique au produit à partir d'une source partagée.
