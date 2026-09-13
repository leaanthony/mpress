---
title: Pourquoi M-Press est rapide
description: Comment un seul processus Go maintient les générations, régénérations, vérifications et pages générées petites et réactives.
translationKey: performance-design
order: 31
---

M-Press considère le temps de retour court comme une fonctionnalité produit. Les auteurs doivent pouvoir
enregistrer le Markdown, inspecter le résultat et continuer à écrire sans attendre que
une chaîne d'outils JavaScript démarre ou régénère une application.

## Un seul processus natif fait le travail

M-Press est un exécutable Go unique. Une génération ne lance pas Node.js, ne résout pas de paquets,
ne transpile pas le code de l'application, ni ne contacte un service réseau. Le même exécutable
charge la configuration, lit le contenu, génère les pages, crée les index de recherche,
optimise les ressources et écrit le site statique.

Cette conception réduit le temps d'installation et supprime la communication entre les processus de génération séparés
processus.

## L'analyse est mise en cache et parallèle

M-Press calcule l'empreinte de chaque page source et stocke sa forme analysée sous
`.mpress/cache/parse/`. Une page inchangée peut réutiliser ce résultat lors de la prochaine
génération. Une page modifiée reçoit une nouvelle empreinte et ne peut pas utiliser du contenu obsolète.

L'analyse Markdown utilise plusieurs workers lorsque la machine dispose de cœurs processeur disponibles
cœurs. Les résultats reviennent dans leur ordre d'origine, donc le travail parallèle ne rend pas
le site généré non déterministe.

## La validation commence pendant la génération

Les vérifications strictes n'ont pas besoin de rouvrir chaque page générée pour découvrir ses liens.
La génération collecte les routes, les liens et les ressources pendant qu'elle rend et copie les fichiers.
L'étape finale de validation résout cet index en mémoire par rapport à la sortie achevée
sortie.

Le menu de développement affiche un diagramme de temps pour chaque étape de vérification. Le diagramme rend
la découverte lente de contenu, l'analyse, le rendu, l'optimisation ou la validation des liens
visible au lieu de masquer le total derrière une seule durée.

## L'optimisation pour la production est intégrée

Les builds de production utilisent le minificateur CSS et JavaScript natif Go de M-Press. Ils
analysent le HTML généré et les scripts d'exécution pour supprimer les règles de classe et d'ID qui sont
manifestement inutiles. Les règles responsives sont filtrées récursivement. Les règles que des composants personnalisés
HTML ou des composants d'exécution peuvent devoir rester intactes.

Cette optimisation n'a pas besoin de paquet Node.js, de lockfile, ni de commande externe.
Passez `--no-purge-css` lorsqu'un projet doit conserver chaque sélecteur.

## Les pages générées restent statiques

Le navigateur reçoit d'abord du HTML statique. La navigation, le contenu, les composants et
les exemples de code n'attendent pas qu'un framework client rende la page. Un petit script ajoute
des fonctionnalités progressives telles que la recherche, les onglets, la sélection de thème et
les préférences d'accessibilité.

La sortie statique permet aussi à l'hôte de mettre en cache les mêmes fichiers pour chaque visiteur. Il
n'y a pas de processus de rendu côté serveur à préchauffer, mettre à l'échelle ou surveiller.

## Mesurez votre projet

Exécutez une génération de production avec une sortie lisible par machine :

```sh
mpress build --strict --json
```

Le résultat indique le nombre de pages, le nombre de fichiers, la durée totale de génération et
diagnostics. Exécutez **Exécutez les vérifications** dans le menu de développement pour inspecter le chronométrage de
chaque étape. Utilisez **l'audit Lighthouse** pour mesurer la page courante sur mobile ou
bureau.

Le temps de génération dépend du nombre de pages, de la taille du contenu, du stockage local, du nombre de processeurs,
et du nombre de ressources statiques. M-Press rapporte les mesures au lieu de
de promettre une seule durée pour chaque projet.
