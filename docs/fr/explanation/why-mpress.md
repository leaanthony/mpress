---
title: Pourquoi M-Press existe
description: Les raisons de conception d'un petit générateur de documentation statique qui utilise Markdown, HTML et un exécutable Go.
order: 30
---

M-Press s'adresse aux équipes qui veulent de la documentation sans application JavaScript
chaîne d'outils. Il accepte Markdown et HTML standard. Il produit des fichiers que n'importe quel
serveur web statique peut publier.

## Le contenu portable est la contrainte principale

La documentation survit généralement à son premier générateur. Les composants spécifiques aux frameworks
rendent la migration coûteuse parce que le contenu devient du code source d'application.
M-Press maintient son modèle de rédaction minimal pour que les équipes puissent lire et transformer la source
avec des outils courants.

Le HTML est la voie de sortie. Les auteurs peuvent utiliser du HTML sémantique quand Markdown n'est pas
suffisant. Une page d'accueil peut utiliser une mise en page personnalisée sans changer le format de
la documentation technique.

## Des builds stricts rendent la migration plus sûre

Un outil de migration ne doit pas masquer le contenu qu'il ne peut pas convertir. L'importateur Starlight de M-Press
convertit la syntaxe prise en charge et écrit un rapport de migration.
Les composants non pris en charge restent visibles et provoquent une erreur de build stricte.

Ce comportement peut faire échouer la première build de migration. L'échec est utile :
il identifie le travail que perdrait une conversion silencieuse.

## Les outils locaux et les services partagés ont des rôles différents

Le binaire M-Press assume le travail local déterministe. Cela inclut l'analyse,
la navigation, la recherche privée, les contrôles d'accessibilité, les routes linguistiques, la
traduction, les instantanés de versions, les vérifications, l'optimisation pour la production et la
sortie statique.

Des services externes facultatifs peuvent coordonner les personnes et les données
hébergées. M-Press génère et publie des sites statiques complets sans abonnement.

Voir la [présentation des fonctionnalités M-Press](/features/) pour le flux complet.
