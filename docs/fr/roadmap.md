---
title: Après le MVP
description: Travail ordonné au-delà du contrat de publication M-Press 0.1 vérifié.
order: 22
---

Le contrat M-Press 0.1 est terminé. La parité des composants est en place pour la
suite de composants conservée, le véritable corpus Wails s'importe et se construit proprement, et
les portes de publication publiques et privées passent. Le chat API reste délibérément
différé. Le travail ci‑dessous renforce le produit sur la voie vers 1.0 ; ce n'est pas un
Il n'est pas bloquant pour la première version MVP.

## 1. Déploiement Wails

La porte de migration côté générateur est complète : les sources Markdown 511 sont publiées en tant que
pages de contenu 501 et fichiers 646, la compilation stricte réussit, et le site généré présente
aucun lien ni ressource cassés. Le travail restant est le déploiement du projet.

- Examinez les constats explicites de migration avec les mainteneurs de Wails.
- Comparez le manifeste de routes généré avec les routes déployées en production.
- Ajoutez les échecs spécifiques au déploiement au corpus de conformité privé.
- Exécutez un déploiement par étapes avant de changer l'origine de la documentation publique.

## 2. Renforcement de la traduction

Le générateur fournit désormais la traduction automatique structurée,
des fournisseurs compatibles OpenRouter et OpenAI, l'application du glossaire, des clés de page stables
clés, empreintes sources, détection des modifications manuelles, traduction de la navigation, et un
flux de développement sécurisé. Le travail restant étend la validation et la revue.

- Faites passer le corpus multilingue Wails par les contrôles des fournisseurs privés et du navigateur.
- Ajoutez un mode de test pseudo-langue pour révéler les chaînes d'interface non traduites et
  les mises en page qui échouent avec des textes plus longs.
- Ajoutez l'import et l'export XLIFF pour les outils de traduction professionnels.
- Mesurez la qualité de traduction avec une revue MQM humaine. Traitez les scores automatisés comme
  des signaux, pas des conditions de publication.

Des services externes facultatifs peuvent fournir une mémoire de traduction partagée, des tâches, des budgets, des
pull requests, des workflows fournisseurs, et l'historique d'audit de l'organisation.

## 3. Renforcement de la gestion des versions

La capture de version, les sommes de contrôle, la liste, la vérification et le montage existent. Ils nécessitent maintenant
des tests de sous-chemin au niveau de déploiement.

- Définir un ordre stable, des libellés d'affichage et des alias tels que `latest`, et le comportement
  lorsque la version courante change.
- Tester les instantanés de version multilingues et les chemins non racine `baseURL` racine.
- Rendez la capture atomique et vérifiez que les captures interrompues ne remplacent jamais un
  artefact.

Le format d'artefact immuable local reste portable. Le stockage géré, les URL de prévisualisation
les URL, la rétention, la promotion et le rollback peuvent utiliser des services externes facultatifs.

## 4. Finitions du développement et du thème

- Ajoutez la planification d'importation, les opérations de renommage et de déplacement, la gestion des assets, l'annulation, et un
  réinitialisation de projet récupérable au protocole d'édition. L'état de construction, la
  barre de développement, formulaires de configuration, intégration, contrôles, rechargement à chaud, et
  sont en place.
- Ajoutez une coloration syntaxique à la compilation sans dépendance d'exécution dans le navigateur.
- Ajoutez des contrôles de copie aux blocs de code balisés ordinaires, pas seulement aux terminaux.
- Ajoutez des contrôles de permalien de titre et rendez les liens profonds évidents au focus ou au survol.
- Ajoutez des données structurées pour les pages spécialisées de référence et de blog.
- Améliorez les erreurs développeur avec des lignes-source, des suggestions, et une page d'erreur
  du serveur de prévisualisation qui ne détruit pas la dernière compilation réussie.
- Terminez les tests responsive, d'impression, de réduction du mouvement, de haut contraste et de contenu long
  pour le catalogue complet de composants.

## 5. Ingénierie des publications

- Produisez des binaires signés pour Linux, macOS et Windows à partir de versions taguées.
- Publiez les sommes de contrôle et un parcours d'installation et de mise à niveau documenté.
- Stabilisez le schéma de configuration, les codes de sortie CLI, les diagnostics JSON, et le
  politique de compatibilité avant d'appeler le format `1.0`.
- Ajoutez des baselines privées de navigateur, des vérifications d'accessibilité, des fixtures de migration, et
  des cas de parseur adversariaux au pipeline de publication.
- Établissez des budgets de performance pour la compilation à froid, la recompilation, le poids des pages générées,
  et la taille de l'index de recherche.
