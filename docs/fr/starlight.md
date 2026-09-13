---
title: Importer depuis Starlight
description: Migrez la documentation Astro Starlight vers M-Press sans masquer le contenu non pris en charge.
order: 11
---

L'importateur convertit les parties d'un projet Starlight qui peuvent rester portables :
pages, frontmatter, navigation, langues, ressources publiques, images sources, réseaux sociaux
liens, image de marque et composants de documentation pris en charge.

```sh
mpress import --from starlight ../starlight-docs --output ./docs
cd docs
mpress build
```

La source peut être la racine d'un projet Starlight ou un répertoire contenant du Markdown.
L'importateur recherche `src/content/docs`, `docs`, puis `src`.

## Rapport de migration

Chaque import écrit `migration-report.md`. Examinez-le avant d'activer les builds stricts
builds. Les constatations peuvent inclure :

- composants MDX personnalisés ou non pris en charge;
- importations JSX spécifiques au projet;
- contenu privé préfixé par un underscore;
- liens de la barre latérale non résolus;
- styles ou composants nécessitant un remplacement statique.

Les composants connus sont convertis en directives M-Press ou en HTML statique. Les composants inconnus
restent visibles dans les pages générées et provoquent l'échec d'un build strict.

## Boucle de migration recommandée

1. Importez dans un nouveau répertoire ; n'écrasez pas le projet d'origine.
2. Lisez `migration-report.md` de haut en bas.
3. Exécutez `mpress build --json` et corrigez les diagnostics d'erreur.
4. Exécutez `mpress check` et classez les liens brisés, les fragments et les ressources locales.
5. Comparez des pages représentatives aux largeurs bureau et mobile.
6. Activez `mpress build --strict` comme condition de publication.

@note{type="tip" title="The importer is an adapter"}
Ne conservez pas la syntaxe du framework uniquement pour assurer une compatibilité parfaite avec la source. Convertissez
la présentation personnalisée importante en Markdown, en HTML ordinaire ou en un petit composant M-Press
afin que la documentation migrée devienne plus simple au fil du temps.
@end
