---
title: Offrir des contrôles d'accessibilité aux lecteurs
description: Permettre à chaque lecteur d'activer des réglages pour la lecture privée, la mise au point, le mouvement, le contraste, les liens et les couleurs.
translationKey: accessibility-controls
order: 7
---

M-Press inclut le menu d'accessibilité dans le thème par défaut. Il est
activé par défaut. Chaque lecteur peut adapter le site sans modifier la source
ou affecter un autre visiteur.

## Activer le menu

Les nouveaux projets contiennent déjà cette configuration :

```yaml
accessibility:
  enabled: true
  shortcut: Mod+A
```

Ajoutez ce même réglage à un projet existant. Ensuite, lancez le serveur de développement :

```sh
mpress dev
```

Ouvrez l'icône de personne dans la barre de navigation. Le panneau regroupe les réglages en
les onglets Lecture, Mise au point et Vision. Appuyez sur Command+A sur un appareil Apple ou Ctrl+A
sur une autre plateforme pour ouvrir le panneau sans quitter le clavier. Le
raccourci ne remplace pas Select All lorsque un champ modifiable a le focus.

## Réglages de lecture

Les lecteurs peuvent :

- augmenter la taille du texte principal ;
- utiliser une famille de polices simple et espacée ;
- ajouter un espacement des lignes, des mots et des caractères ;
- mettre en valeur le début des mots longs avec bionic reading.

Ces modifications s'appliquent au contenu principal. Elles n'agrandissent pas la navigation ni les
contrôles au point de les rendre difficilement utilisables.

## Réglages de mise au point

Les lecteurs peuvent :

- assombrir la navigation tant que le curseur n'y est pas ;
- suivre le pointeur avec un guide de lecture horizontal ;
- désactiver les animations non essentielles et le défilement fluide.

Le thème généré respecte aussi les préférences reduced-motion du navigateur et
reduced-transparency.

## Réglages de vision

Les lecteurs peuvent :

- augmenter le contraste du texte et des bordures ;
- souligner les liens pour que la couleur ne soit pas le seul signal visuel ;
- choisir un profil de distinction rouge/vert ;
- choisir un profil de distinction bleu/jaune ;
- réduire la saturation des couleurs.

Le thème prend en charge le mode forced-colour et la préférence increased-contrast du
navigateur. Les modes clair et sombre reçoivent des valeurs de couleur adaptées.

## Confidentialité et persistance

M-Press enregistre les choix du lecteur dans le stockage local du navigateur. Il restaure la
les paramètres visuels avant le rendu de la feuille de style principale. Cela évite un clignotement de la
mauvaise taille de texte, du contraste ou du profil de couleur.

Le site généré n'envoie pas les choix d'accessibilité à M-Press ni à un autre
service. Le bouton **Reset settings** supprime tous les choix enregistrés.

## Tester le résultat

Testez le site au clavier et à une largeur mobile. Vérifiez que :

1. le bouton d'accessibilité porte le nom **Accessibility settings**;
2. chaque onglet et contrôle peut recevoir le focus clavier ;
3. les réglages sélectionnés restent après un rechargement ;
4. **Reset settings** restaure le thème par défaut ;
5. la page reste lisible en mode forced-colour et reduced-motion.

Utilisez **Lighthouse audit** dans le menu de développement pour une vérification automatisée supplémentaire
d'accessibilité. Les contrôles automatisés ne remplacent pas les tests au clavier et
avec des technologies d'assistance.

Définissez `accessibility.enabled: false` uniquement lorsque le projet doit fournir une
interface d'accessibilité. Cela supprime le bouton, le panneau, les règles et le script de
le site généré.
