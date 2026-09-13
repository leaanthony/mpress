---
title: Composants
description: Une bibliothèque de composants statique et accessible pour de la documentation sérieuse.
order: 6
---

M-Press livre la bibliothèque complète de composants dans le binaire principal. Chaque
composant est rendu au moment de la génération ; JavaScript ajoute seulement un enrichissement progressif.

## Une grammaire de composants

Chaque composant bloc commence par `@name{attributes}` et se termine par `@end`.
Les composants imbriqués utilisent la même grammaire à tous les niveaux. Les composants feuilles tels que
les boutons et les images peuvent utiliser leurs formes compactes sur une seule ligne.

M-Press ne considère comme directive qu'un nom de composant bloc enregistré en début de
ligne. Les boutons et images compacts doivent correspondre exactement à leurs signatures
complètes, ainsi les adresses e‑mail et les mentions ordinaires restent inchangées.

## Remarques et détails

@note{type="tip" title="Plain Markdown inside"}
Utilisez **Markdown**, des liens, des listes et du code dans une note. Les types sont `info`, `tip`,
`warning`, `caution`, `danger`, et `important`.
@end

@details{title="Source"}
```md
@note{type="tip" title="Plain Markdown inside"}
Utilisez **Markdown**, des liens, des listes et du code dans une note. Les types sont `info`, `tip`,
`warning`, `caution`, `danger`, et `important`.
@end
```
@end

## Onglets

@tabs
[Go]
Exécuter `go test ./...`.

[Shell]
Exécuter `mpress build --strict`.
@end

@details{title="Source"}
```md
@tabs
[Go]
Exécuter `go test ./...`.

[Shell]
Exécuter `mpress build --strict`.
@end
```
@end

Les onglets utilisent des boutons, des panneaux d'onglets et des états ARIA. Le premier panneau reste lisible
lorsque les scripts sont indisponibles.

## Cartes et liens

@cards{cols="2"}
[Démarrer un projet](/getting-started/)
Installez un seul binaire et générez un site.

---

[Configurer M-Press](/configuration/)
Définissez la navigation, les langues, les versions et les options de thème.
@end

@linkcard{title="Read the authoring guide" href="/authoring/" description="Markdown, HTML, assets, and front matter." icon="→"}

@details{title="Source"}
```md
@cards{cols="2"}
[Démarrer un projet](/getting-started/)
Installez un seul binaire et générez un site.

---

[Configurer M-Press](/configuration/)
Définissez la navigation, les langues, les versions et les options de thème.
@end

@linkcard{title="Read the authoring guide" href="/authoring/" description="Markdown, HTML, assets, and front matter." icon="→"}
```
@end

## Étapes

@steps
### Rédiger
Ajoutez du Markdown ou du HTML à `docs/`.

### Aperçu
Exécutez `mpress dev` et modifiez avec rechargement à chaud.

### Publier
Exécutez `mpress build --strict` et déployez `site/`.
@end

@details{title="Source"}
```md
@steps
### Rédiger
Ajoutez du Markdown ou du HTML à `docs/`.

### Aperçu
Exécutez `mpress dev` et modifiez avec rechargement à chaud.

### Publier
Exécutez `mpress build --strict` et déployez `site/`.
@end
```
@end

## Arborescences, badges et boutons

@filetree
docs/
  index.md  Page d'accueil
  components.md  Ce catalogue de composants
mpress.yaml  Configuration du site
@end

Utilisez {badge.success:stable} pour un statut compact, ou
@button[Open the guide](/getting-started/){secondary} for a clear
une action dans le texte.

@details{title="Source"}
```md
@filetree
docs/
  index.md  Page d'accueil
  components.md  Ce catalogue de composants
mpress.yaml  Configuration du site
@end

Utilisez {badge.success:stable} pour un statut compact, ou
@button[Open the guide](/getting-started/){secondary} for a clear
une action dans le texte.
```
@end

## Conteneurs de mise en page

Les conteneurs ajoutent de petites règles de mise en page autorisées sans transformer le Markdown
fichier en un langage de gabarits.

@container{display=grid|columns=2|gap=1rem}
@tip[Static]
La mise en page complète existe dans le HTML généré.
@end
@info[Responsive]
Le thème par défaut compresse les mises en page denses sur les petits écrans.
@end
@end

@details{title="Source"}
```md
@container{display=grid|columns=2|gap=1rem}
@tip[Static]
La mise en page complète existe dans le HTML généré.
@end
@info[Responsive]
Le thème par défaut compresse les mises en page denses sur les petits écrans.
@end
@end
```
@end

## Mises en page des pages d'atterrissage

Les pages d'atterrissage peuvent utiliser des directives de mise en page Markdown imbriquées. Utilisez `section`,
`columns`, et `column` pour définir la structure de la page. Utilisez `actions` pour un groupe
de boutons. Utilisez `headline` lorsqu'un grand titre nécessite des sauts de ligne délibérés.

```md
@section{variant=hero}
@columns{variant=hero}
@column{variant=hero-copy}
@headline
Modern docs.
Rich components.
Just Markdown.
@end

Write the supporting copy as ordinary Markdown.

@actions
@button[Start the tutorial](/tutorials/first-site/){primary}
@button[Read the guide](/authoring/){secondary}
@end
@end

@column
Add a product example, image, terminal, or another component here.
@end
@end
@end
```

Le thème par défaut fournit des styles responsives pour les variantes de page d'atterrissage nommées.
Tout le contenu de mise en page est rendu au moment de la génération. La source ne requiert pas HTML,
MDX, ni un framework JavaScript.

## Compatibilité

Compatible avec Starlight `Aside`, `Tabs`, `TabItem`, `CardGrid`, `Card`, `LinkCard`,
`Steps`, `FileTree`, `Badge`, et `Image` la syntaxe est comprise pour les migrations.
Le nouveau contenu devrait utiliser les `@` directives car elles sont portables et faciles
à lire.
