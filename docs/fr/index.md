---
title: Documentation M-Press
description: Créez une documentation moderne et riche avec Markdown, pas MDX.
layout: landing
translationKey: mpress-home
order: 1
---

@section{variant=hero}
@columns{variant=hero}
@column{variant=hero-copy}
@headline
Documentation moderne.
Composants riches.
Juste Markdown.
@end

M-Press vous fournit des composants riches, une recherche privée, des contrôles d'accessibilité pour le lecteur, des versions publiées, et une traduction assistée par IA sécurisée dans un seul binaire Go rapide. Le noyau complet de documentation statique est gratuit et sous licence MIT. Pas de Node.js. Pas de MDX. Pas de framework.

@actions
@button[Build your first site](/tutorials/first-site/){primary}
@button[See what is included](/features/){secondary}
@end
@end

@column{variant=site-screenshot|class=mp-home-site-screenshot-hero}
@image{light="/images/mpress-docs-light.png" dark="/images/mpress-docs-dark.png" alt="The generated M-Press tutorial site"}
@end
@end
@end

@section{variant=story|class=mp-home-story-stack}
@columns{variant=story}
@column{variant=story-copy}
## Ne maintenez pas un frontend juste pour publier du texte.

La documentation ne doit pas commencer par un gestionnaire de paquets, une mise à jour de framework et un conflit de lockfile. M-Press maintient le modèle d'écriture délibérément minimal : un binaire Go, un fichier YAML et du Markdown ordinaire.

- **Pas de runtime Node.js.** Installez un exécutable natif localement et en CI.
- **Pas de couplage MDX.** Vos sources restent lisibles hors de M-Press.
- **Pas de graphe de build caché.** Le résultat généré est un HTML statique portable.

@button[See the complete feature reference](/features/){secondary}
@end

@column{variant=story-visual|class=mp-home-stack-visual}
### Un projet clair.

~~~yaml
site:
  title: Product documentation
  content: docs
  output: site
accessibility:
  enabled: true
languages:
  - code: en
    default: true
~~~

**`mpress.yaml` + Fichiers Markdown + un exécutable**
@end
@end
@end

@section{variant=story|class="mp-home-story-components mp-home-story-reversed"}
@columns{variant=story}
@column{variant=story-copy}
## Documentation riche sans transformer chaque page en application.

Les lecteurs attendent encore la recherche, les onglets, les étapes, les terminaux, les diagrammes, les annotations de code, les arbres de fichiers et une navigation responsive. M-Press les construit à partir de directives Markdown simples et livre le comportement avec le site.

- **La recherche est automatique et privée.** M-Press construit un index statique pour chaque langue. Les requêtes restent dans le navigateur.
- **Les composants sont inclus.** Les auteurs n'installent pas de paquet pour chaque modèle.
- **La sortie fonctionne sans runtime de framework.** Hébergez-le partout où l'on sert des fichiers statiques.

@button[Explore every component](/components/){secondary}
@end

@column{variant=story-visual|class=mp-home-story-image}
@image{light="/images/mpress-docs-light.png" dark="/images/mpress-docs-dark.png" alt="An M-Press documentation page with navigation, search, a terminal, and a page outline"}
@end
@end
@end

@section{variant=story|class=mp-home-story-speed}
@columns{variant=story}
@column{variant=story-copy}
## Continuez à écrire pendant que le site suit.

Les builds lents brisent la concentration. M-Press exécute un serveur de développement qui surveille votre contenu, reconstruit automatiquement et rafraîchit le navigateur. L'analyse parallèle, les caches persistants et la collecte de liens pendant le rendu ôtent le travail répétitif du chemin critique.

- **Retour immédiat.** Enregistrez un fichier et voyez la page générée se mettre à jour.
- **Indicateur visible.** Le menu de développement indique la durée de chaque étape de build et de validation.
- **Qualité dans le même flux de travail.** Exécutez des contrôles stricts, inspectez les liens et les ressources, et lancez un audit Lighthouse avant la publication.

@button[See how M-Press stays fast](/explanation/performance/){secondary}
@end

@column{variant=story-visual|class=mp-home-speed-visual}
### Build de développement

**Prêt · reconstruction automatique terminée**

@capabilities
- **Analyser le Markdown** Mis en cache et parallèle
- **Rendre les pages et la recherche** Générés ensemble
- **Valider les liens et les ressources** Collectés pendant le rendu
- **Optimiser la sortie** Minification native du CSS et du JavaScript
@end

**Lighthouse intégré.**

Mesurez les performances, l'accessibilité, les bonnes pratiques et le SEO. Utilisez les résultats pour pousser chaque score vers 100.
@end
@end
@end

@section{variant=story|class="mp-home-story-accessibility mp-home-story-reversed"}
@columns{variant=story}
@column{variant=story-copy}
## L'accessibilité n'est pas une réflexion secondaire ni une option payante.

Une seule présentation ne convient pas à tous les lecteurs. Le menu d'accessibilité optionnel est activé par défaut et stocke les préférences de chaque visiteur dans son navigateur. Les propriétaires du site ne reçoivent pas de données de santé ou de préférences.

- **Contrôles de lecture.** Modifiez la taille du texte, la police, l'espacement et l'emphase des mots.
- **Contrôles de mise au point.** Atténuez les distractions, suivez un guide de lecture et réduisez les animations.
- **Contrôles visuels.** Augmentez le contraste, soulignez les liens et sélectionnez des profils de couleurs.

@button[Use the accessibility controls](/accessibility/){secondary}
@end

@column{variant=site-screenshot|class=mp-home-accessibility-screenshot}
@image{light="/images/mpress-accessibility-light.png" dark="/images/mpress-accessibility-dark.png" alt="Le menu d’accessibilité en anglais, avec tous les réglages de lecture" expand}
@end
@end
@end

@section{variant=story|class=mp-home-story-global}
@columns{variant=story}
@column{variant=story-copy}
## Les langues et les versions appartiennent au cœur.

Les traductions deviennent risquées lorsque les blocs de code, la syntaxe des composants et les liens sont traités comme du texte ordinaire. Les versions deviennent confuses lorsque d'anciennes pages changent discrètement. M-Press prend en charge ces deux problématiques directement.

- **Traduction sécurisée.** Protégez la structure, utilisez des glossaires et des guides de style, et suivez le texte obsolète, manuel, en conflit et révisé.
- **Sites de langue indépendants.** Générez les routes, la navigation et la recherche pour chaque langue.
- **Versions immuables.** Capturez, calculez le checksum, vérifiez et montez des releases complètes avec un sélecteur visible par le lecteur.

@button[Read about translation](/translation/){secondary}
@button[Read about versioning](/versioning/){secondary}
@end

@column{variant=story-visual|class=mp-home-global-visual}
### Couverture de la documentation

@capabilities
- **Anglais** Actuel · toutes les pages
- **Français** Révisé · toutes les pages
- **Allemand** 14 pages nécessitent une relecture
- **Japonais** Traduction en cours
@end

### Versions publiées

`next` · `v3.0` · `v2.10`
@end
@end
@end

@section{variant=story|class="mp-home-story-custom mp-home-story-reversed"}
@columns{variant=story}
@column{variant=story-copy}
## Commencez avec un thème complet. Personnalisez-le pour qu'il vous ressemble.

Un générateur de documentation ne doit pas imposer la même identité visuelle à tous les produits. Configurez la couleur, la typographie, les logos clair et sombre, la navigation, les liens sociaux, la présentation du blog et les valeurs d'accessibilité depuis un seul fichier de projet. Ajoutez du CSS personnalisé lorsque le design doit aller plus loin.

- **Adaptatif par défaut.** La navigation, le contenu, les composants et les contrôles s'adaptent aux petits écrans.
- **Modes clair et sombre.** Utilisez des ressources de marque et des valeurs de thème séparées si nécessaire.
- **Sortie statique.** Déployez sur Cloudflare, GitHub Pages, un stockage d'objets ou tout serveur web.

@button[Configure a site](/configuration/){secondary}
@end

@column{variant=story-visual|class=mp-home-custom-visual}
@image{light="/images/mpress-home-mobile-light.png" dark="/images/mpress-home-mobile-dark.png" alt="The responsive M-Press landing page on a mobile screen"}

### Votre marque, à toute échelle

Le même contenu Markdown peut supporter une page produit complète et une expérience de lecture mobile ciblée.
@end
@end
@end

@section{variant=migration}
@columns{variant=migration}
@column
## Conservez le contenu. Supprimez la chaîne d'outils JavaScript.
@end

@column
L'importateur transfère les pages, les composants pris en charge, la navigation, les langues, le frontmatter, les assets publics et la configuration. Il écrit un rapport pour chaque modèle qui nécessite une décision humaine.

@button[Read the migration guide](/starlight/){secondary}
@end
@end
@end

@section{variant=resources}
@column{variant=heading|as=header}
## Choisissez une tâche.
@end

@resources
[01 · Tutoriel](/tutorials/first-site/)
## Construisez votre premier site
Commencez avec un répertoire vide et créez une documentation fonctionnelle.
---
[02 · Guide pratique](/how-to/custom-landing-page/)
## Créez une page d'accueil en Markdown
Créez une page produit complète avec des directives de mise en page Markdown.
---
[03 · Guide pratique](/translation/)
## Ajoutez une langue
Créez des itinéraires traduits et gérez les pages manquantes.
---
[04 · Guide pratique](/versioning/)
## Publiez la documentation versionnée
Capturez et vérifiez des instantanés statiques de publication.
@end
@end

@section{variant=final}
@column
## Construisez une documentation riche sans Node.

Créez un projet M-Press fonctionnel. L'accessibilité, les versions, les traductions, la recherche, les vérifications, les composants et l'optimisation pour la production sont prêts dès la première génération.
@end

@actions
@button[Start the tutorial](/tutorials/first-site/){primary}
@button[View the source](https://github.com/leaanthony/mpress){secondary}
@end
@end
