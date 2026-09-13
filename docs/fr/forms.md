---
title: Formulaires Markdown
description: Créez des formulaires accessibles à partir de Markdown ordinaire à l'intérieur d'un bloc de formulaire M-Press.
order: 7
---

Les formulaires M-Press sont des documents Markdown avec des contrôles. Le bloc de formulaire établit
le contexte d'analyse. Les titres, les explications, les listes et les contrôles peuvent être mélangés
dans l'ordre qui a du sens pour la personne qui remplit le formulaire.

## Un formulaire complet

```md
@form[config.save]

# Edit this site

Change the identity and appearance of this documentation site.

## Site identity

Site title*: [text](site.title)

Description: [textarea](site.description){rows=4}

Public URL: [url](site.url)

## Appearance

Colour scheme: [select](theme.mode)
- Use the device setting = system
- Light
- Dark

[ ] Enable accessibility controls (accessibility.enabled)

[Save and rebuild](submit)

@end
```

Le libellé vient en premier. La déclaration du contrôle suit les deux-points. Un astérisque
indique un champ obligatoire. Le nom entre parenthèses est la liaison de valeur utilisée par
l'action du formulaire.

## Contrôles

Utilisez les noms d'entrée HTML courants : `text`, `url`, `email`, `number`, `date`,
`color`, `password`, `search`, `file`, `range`, et `tel`. Utilisez `textarea` pour
texte long et `select` pour une liste d'options.

```md
Email address*: [email](contact.email)

Age: [number](profile.age){min=18 max=120}

Biography: [textarea](profile.biography){rows=6}

Search shortcut: [shortcut](search.shortcut){placeholder=Mod+K}
```

Dans l'éditeur des paramètres de développement, `shortcut` les champs capturent la combinaison de touches suivante
au lieu d'accepter du texte libre. L'éditeur écrit des
valeurs portables telles que `Mod+K`, prend en charge `None` en appuyant sur Backspace, et signale
les conflits avec d'autres raccourcis M-Press avant l'enregistrement du formulaire.

Les options utilisent des éléments de liste Markdown ordinaires. Si le libellé visible et la valeur soumise
diffèrent, séparez-les par `=`.

```md
Country: [select](profile.country)
- Australia = au
- France = fr
- United Kingdom = gb
```

Les cases à cocher utilisent la notation familière des listes de tâches :

```md
[ ] Enable search (features.search)

[x] Enable accessibility controls (features.accessibility)
```

Pour un groupe de choix, utilisez `radio` ou `checkboxes` et suivez-le d'une liste.

```md
Colour scheme: [radio](theme.mode)
- (x) System = system
- ( ) Light = light
- ( ) Dark = dark
```

## Actions et mise en page

Utilisez `[Button text](submit)` pour l'action principale du formulaire et
`[Reset](reset)` pour restaurer les valeurs initiales du navigateur. M-Press rend le
formulaire avec des libellés, des ID stables, l'état requis, le focus clavier et un empilement
réactif. Les liens Markdown normaux et les listes de tâches en dehors d'un bloc de formulaire conservent leur
sens habituel.

Le formulaire peut contenir du Markdown normal et des composants de mise en page M-Press. Cela permet de
garder lisibles les longs flux de paramètres au lieu de transformer la source en un
langage de schéma.

```md
@form[contact.send]

## Tell us about the project

The more context you provide, the more useful the reply will be.

Project name*: [text](project.name)

What are you building?: [textarea](project.summary){rows=8}

[Send message](submit)

@end
```
