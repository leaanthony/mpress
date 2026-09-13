---
title: Créer une page d'atterrissage Markdown
description: Créez une page d'atterrissage complète avec des directives de mise en page Markdown et conservez l'en-tête commun du site.
order: 6
---

Utilisez une mise en page d'atterrissage lorsqu'une page nécessite un design spécifique au produit. La mise en page
conserve l'en-tête, la recherche, le sélecteur de langue, le sélecteur de version et le contrôle du thème.
Elle supprime la barre latérale de la documentation, la table des matières, le titre de page automatique,
et les liens précédent et suivant.

## Définir la mise en page d'atterrissage

Ajoutez `layout: landing` au frontmatter de la page :

```md
---
title: Product documentation
description: Build with the product.
layout: landing
---

@section{variant=hero}
@columns{variant=hero}
@column{variant=hero-copy}
@headline
Build with the product.
Start with confidence.
@end

Create complete documentation from plain Markdown.

@actions
@button[Start the tutorial](/tutorials/first-site/){primary}
@end
@end

@column
Add a screenshot, terminal, or documentation preview here.
@end
@end
@end
```

La source de la page reste en Markdown. Les directives ajoutent des zones de mise en page sémantiques à
l'HTML généré. M-Press rend la page complète avant qu'elle n'atteigne le
navigateur.

## Ajouter des styles

Créez une feuille de style telle que `landing.css`. Ensuite, configurez-la :

```yaml
build:
  customCSS: landing.css
```

Utilisez des sélecteurs spécifiques à la page pour empêcher que les styles d'atterrissage modifient les pages
de documentation normales :

```css
.landing-page .mpress-section-hero {
  max-width: 72rem;
  margin: 0 auto;
  padding: 8rem 1.5rem;
}
```

La feuille de style personnalisée se charge après le thème par défaut.

## Vérifier le résultat

Vérifiez ces conditions sur les écrans de bureau et mobiles :

1. Le focus clavier suit l'ordre visuel.
2. La structure des titres commence par un `h1` élément.
3. Le texte et les contrôles ont un contraste suffisant.
4. La page ne provoque pas de défilement horizontal.
5. La page reste utilisable lorsque JavaScript est désactivé.

Exécutez `mpress build --strict` et `mpress check` avant de publier la page.

La même page d'atterrissage Markdown doit s'adapter aux écrans étroits :

![La page d'atterrissage de la documentation M-Press à la largeur mobile](/images/mpress-home-mobile.png)
