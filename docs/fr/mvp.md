---
title: M-Press 0.1 MVP
description: Le contrat de publication volontairement limité pour la première version de M-Press.
order: 21
---

## Résultat

M-Press 0.1 réussit lorsque Wails peut remplacer son Starlight build par un
Go binary while authors continue to write Markdown, ordinary HTML, and a small
ensemble documenté de composants de contenu portables.

C'est un générateur de documentation, pas un framework d'application web général.

## Statut de vérification

Le contrat de publication 0.1 est mis en œuvre et testé à la fois contre le public
suite de tests et la suite de conformité privée. Le véritable corpus de documentation Wails
contient des sources Markdown 511, publie des pages de contenu 501 et des fichiers 646, se construit en
mode strict, et passe le vérificateur de liens et d'assets générés. Les vérifications sur navigateur desktop et 375px
couvrent la page d'accueil, le shell de documentation, la recherche, les contrôles d'accessibilité
les contrôles, les outils du projet, l'onboarding et la configuration en modes sombre et clair.

## Contrat de publication

### Rédaction et build

- Un exécutable pour Linux, macOS et Windows.
- `mpress init`, `dev`, `build`, `clean`, `check`, et `deploy`.
- CommonMark plus tableaux, listes de tâches, notes de bas de page, IDs de titres, et HTML ordinaire.
- YAML frontmatter avec routes stables, brouillons, titres, descriptions, et ordonnancement.
- Builds hors ligne déterministes sans runtime Node.js ni exigence réseau.
- Le MDX inconnu ne disparaît jamais : les builds normaux affichent un marqueur évident et
  les builds stricts échouent avec un diagnostic au niveau du fichier.

### Expérience de la documentation

- Un thème par défaut soigné et responsive, modes couleur clair/sombre/système, navigation,
  table des matières, liens précédent/suivant, et repères clavier accessibles.
- Recherche client statique avec un index par langue.
- Notes, onglets, cartes, cartes de lien, étapes, arborescences de fichiers, badges, et images.
- CSS personnalisé pour les projets qui ont besoin de leur propre identité visuelle.
- Un menu de projet réservé au développement, une configuration guidée, une configuration structurée,
  préparation du dépôt et de la branche de travail, un panneau d'aperçu en direct déplaçable,
  vérifications de release, aperçus directs Cloudflare, et rechargements automatiques du navigateur.

### Traduction

- Liste explicite de langues et routes préfixées par langue stables.
- La langue par défaut peut se trouver à `/`.
- Le sélecteur de traduction affiche la disponibilité et renvoie les traductions manquantes vers
  la page correspondante dans la langue par défaut.
- Les pages manquantes ne sont pas dupliquées silencieusement dans la sortie traduite.
- Navigation et index de recherche par langue.
- Traduction automatique structurée via OpenRouter, OpenAI, un fournisseur compatible
  fournisseur, ou une commande locale Codex/Claude Code authentifiée sans
  changer le code, les liens, la syntaxe Markdown, ou les composants.
- Un assistant de traduction axé sur le résultat pour ajouter, mettre à jour, et réviser
  les langues sans exposer les détails du fournisseur comme flux de travail principal.
- État de fraîcheur local, édition manuelle et détection de conflits, vérifications de glossaire,
  état de révision de page, et navigation traduite.

Mémoire de traduction partagée, affectations, budgets, pull requests, flux de travail fournisseurs
et l'historique d'audit de l'organisation sont des fonctionnalités de service.

### Versionnage

- Capturer un build statique complété sous une étiquette.
- Un manifeste de sommes de contrôle rend les versions capturées détectables en cas de falsification.
- Vérifier, lister, supprimer, et monter les versions capturées dans des builds ultérieurs.
- Les artefacts de version restent des fichiers statiques déployables et n'exigent jamais le service.
- La sortie générée inclut des règles neutres vis-à-vis de l'hôte `_headers` : les instantanés de version
  sont cacheables immuables, les assets utilisent stale-while-revalidate, et le HTML reste
  revalidable.

### Déploiement

- Publier la sortie statique vérifiée vers Cloudflare Pages ou Netlify depuis le CLI
  ou le menu de développement.
- Les déploiements de prévisualisation sont des brouillons isolés. La production requiert un flag explicite
  et un identifiant conservé hors du projet.
- L'export ZIP reste disponible pour tout autre hébergement statique.

Stockage d'artefacts géré, politiques de rétention, automatisation des releases, URLs de prévisualisation,
et un tableau de bord de version sont des fonctionnalités de service.

### Migration Starlight

- Importer les pages, les composants pris en charge, la navigation, les langues, le frontmatter public
  assets, et les paramètres de présentation pertinents.
- Générer un rapport de migration pour chaque motif nécessitant une revue humaine.
- Valider l'importateur à la fois contre des fixtures privés sélectionnés et le vrai Wails
  corpus de documentation.

## Délibérément hors de 0.1

- Exécution arbitraire de composants Astro/React/Vue/Svelte.
- Un runtime de plugin, une marketplace de thèmes, ou un langage de template général.
- Blog, ecommerce, authentification, rendu côté serveur, et état applicatif.
- Hébergement Git intégré, historique de déploiement géré, analytics, ou fournisseurs de traduction.
- Compatibilité avec chaque plugin Starlight.

## Conditions de publication

1. Les tests unitaires publics et end-to-end sur fixtures passent.
2. Les corpus privés de conformité et de migration passent contre le binaire de release.
3. Une importation Wails se construit sans échec du parseur ; les constats de migration intentionnels
   sont revus et documentés.
4. Les pages générées passent les vérifications de liens/assets et les smoke tests navigateur sur desktop et
   largeurs mobiles.
5. Une version capturée se vérifie et se monte dans un build propre.

Les cinq critères sont désormais couverts. Les constats de migration nécessitant un jugement éditorial
restent explicites dans `migration-report.md`; ils ne sont pas silencieusement
rejetés par l'importateur.
