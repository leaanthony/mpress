---
title: Code et terminal
description: Commandes, explications de code source, diff et points de terminaison d'API.
order: 7
---

## Terminal

Le terminal distingue les commandes, les commentaires et la sortie. Définissez le `prompt`
et `comment` caractères dans les métadonnées. Le bouton de copie copie les
commandes exécutables uniquement. Il ne copie pas les caractères d'invite, les lignes de commentaire complètes, ni
la sortie de commande. Définissez `title` et `frame` (`macos`, `windows`, `linux`, ou
`plain`) comme métadonnées.

@terminal{title="Build the documentation" frame="macos" prompt="$" comment="#"}
# Vérifiez le site avant la génération de production.
$ mpress check
Aucun problème trouvé.
$ mpress build --strict
15 pages générées en 24ms.
@end

@details{title="Source"}
```md
@terminal{title="Build the documentation" frame="macos" prompt="$" comment="#"}
# Vérifiez le site avant la génération de production.
$ mpress check
Aucun problème trouvé.
$ mpress build --strict
15 pages générées en 24ms.
@end
```
@end

## Diffs

@diff{title="mpress.yaml" mode="inline"}
colorScheme: light
search: false
---
colorScheme: system
search: true
@end

@details{title="Source"}
```md
@diff{title="mpress.yaml" mode="inline"}
colorScheme: light
search: false
---
colorScheme: system
search: true
@end
```
@end

Utilisez la valeur par défaut `mode="side-by-side"` pour une comparaison en deux colonnes.

## Code expliqué

@explained
```go
func main() { // (1)
    site.Build() // (2)
}
```

(1) Le programme commence par un point d'entrée Go classique.

(2) Un appel transforme l'arborescence de contenu en site statique.
@end

@details{title="Source"}
````md
@explained
```go
func main() { // (1)
    site.Build() // (2)
}
```

(1) Le programme commence par un point d'entrée Go classique.

(2) Un appel transforme l'arborescence de contenu en site statique.
@end
````
@end

Pointez ou mettez le focus sur une ligne de code numérotée pour ouvrir son explication. Sélectionnez la ligne
sur un écran tactile.

## Points de terminaison de l'API

@api{method="POST" path="/v1/builds"}
Crée une génération de documentation.

| Champ | Type | Obligatoire |
| --- | --- | --- |
| `ref` | string | oui |
| `strict` | boolean | non |
@end

@details{title="Source"}
```md
@api{method="POST" path="/v1/builds"}
Crée une génération de documentation.

| Champ | Type | Obligatoire |
| --- | --- | --- |
| `ref` | string | oui |
| `strict` | boolean | non |
@end
```
@end
