---
title: Architecture
description: Comment le petit exécutable M-Press en Go transforme le contenu local en sortie statique déterministe.
order: 20
---

M-Press est un module Go et un exécutable. La direction des dépendances reste
orientée vers l'intérieur et volontairement simple :

```text
CLI -> config -> discovery -> Markdown/components -> navigation -> static output
                                                        |              |
                                                        +-- search     +-- check
                                                        +-- versions   +-- dev server
```

Le générateur assure la transformation déterministe des fichiers locaux. Les sites générés
contiennent du HTML, du CSS, du JSON, des ressources et un petit script d'amélioration progressive.
Le JavaScript n'est pas requis pour lire ou naviguer dans la documentation.

L'importateur Starlight est un adaptateur à la frontière. Il convertit la syntaxe connue
au format d'auteur portable de M-Press et signale tout le reste.
Le support d'une migration ne doit pas complexifier indéfiniment le parseur principal.

## Périmètre de test

Le dépôt public contient des tests représentatifs pour rassurer les contributeurs.
Le dépôt privé `mpress-tests` teste le binaire compilé en utilisant des
jeux de test, corpus de migration, entrées adverses, et finalement des références visuelles.
Les versions exigent les deux ; la CI publique peut être activée en définissant la variable du dépôt
`PRIVATE_CONFORMANCE=true` et le secret `MPRESS_TESTS_TOKEN`.
