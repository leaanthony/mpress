---
title: Navigation
description: Créez une navigation de documentation explicite ou générée.
order: 7
---

Sans fichier de navigation, M-Press crée un arbre de navigation déterministe à partir de
les pages découvertes. Pour un manuel organisé, ajoutez `_nav.yaml` à la racine du contenu.

```yaml
- label: Home
  link: /
- label: Get started
  items:
    - label: Installation
      link: /installation/
    - label: Configuration
      link: /configuration/
- label: API
  collapsed: true
  autogenerate:
    directory: reference
- label: GitHub
  link: https://github.com/example/project
```

Les éléments peuvent être des liens, des groupes imbriqués ou des répertoires générés automatiquement. Définissez
`collapsed: true` lorsqu'un groupe doit être fermé initialement.

Les builds stricts valident chaque lien de navigation interne par rapport à la
langue par défaut ou à la traduction en cours. Les liens HTTP externes restent inchangés.

Pour une navigation traduite, placez un autre `_nav.yaml` dans le répertoire de langue
répertoire. Par exemple, utilisez `content/fr/_nav.yaml`". S'il est absent, M-Press utilise la
navigation par défaut et achemine correctement les pages traduites disponibles.
