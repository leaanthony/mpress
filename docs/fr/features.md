---
title: Tout ce qu'un site de documentation doit faire
description: Rédaction riche, contrôles de lecture, langues, versions, recherche, vérifications et déploiement dans un seul flux rapide.
layout: landing
translationKey: mpress-features
order: 5
---

@section{variant=hero}
@columns{variant=hero}
@column{variant=hero-copy}
@headline
Écrivez en Markdown.
Publiez un produit complet.
@end

M-Press transforme le Markdown ordinaire en documentation rapide, recherchable et accessible. Les composants riches, traductions, versions, contrôles et déploiements fonctionnent ensemble dès le premier build.

@actions
@button[Créer votre premier site](/tutorials/first-site/){primary}
@button[Explorer les composants](/components/){secondary}
@end
@end

@column{variant=story-visual}
@terminal{title="Un build de production complet" language=bash frame=macos}
$ mpress build --strict
511 pages analysées
8 index de recherche générés
Liens et ressources vérifiés
Site statique généré en 3,2 s
@end
@end
@end
@end

@section{variant=story}
@columns{variant=story}
@column{variant=story-copy}
## Documentation riche sans projet frontend.

Utilisez des mots-clés courts et lisibles pour créer des terminaux, étapes, notes, onglets, arbres de fichiers, diffs, code annoté, tables, vidéos et vues API.

@button[Voir tous les composants](/components/){secondary}
@end
@column{variant=story-visual}
@preview-tabs
[Terminal]
@terminal{title="Démarrer le serveur" language=bash frame=macos}
$ mpress dev
Prêt sur http://127.0.0.1:4174
@end
[Étapes]
@steps
### Écrire normalement
Le contenu reste lisible dans chaque éditeur.
### Construire avec confiance
M-Press vérifie le site avant sa publication.
@end
@end
@end
@end
@end

@section{variant=story}
@columns{variant=story}
@column{variant=story-copy}
## Chaque lecteur adapte l'expérience.

Le menu d'accessibilité est présent dans la barre de navigation. Les préférences restent dans le navigateur du lecteur.

- Taille du texte et largeur de lecture.
- Mise en page pleine largeur ou largeur fixe.
- Police, espacement, contraste, liens et profils de couleur.
- Mouvement réduit, mode focus et guide de lecture.

@button[Ouvrir le menu d'accessibilité](#){primary|action=accessibility}
@end
@column{variant=story-visual}
@preview-tabs
[Lecture]
Taille du texte, mise en page du site et largeur de la colonne.
[Focus]
Réduire les distractions, suivre la lecture et limiter le mouvement.
[Vision]
Augmenter le contraste, souligner les liens et choisir un profil de couleur.
@end
@end
@end
@end

@section{variant=story}
@columns{variant=story}
@column{variant=story-copy}
## Le serveur de développement est votre centre de contrôle.

Configurez le YAML complet, lancez les vérifications, consultez les temps de build, exécutez les audits Lighthouse, exportez un ZIP, traduisez, créez des versions et déployez.

@button[Comprendre les performances](/explanation/performance/){secondary}
@end
@column{variant=story-visual}
@image{light="/images/mpress-config-light.png" dark="/images/mpress-config-dark.png" alt="Configuration structurée d'un projet M-Press"}
@end
@end
@end
