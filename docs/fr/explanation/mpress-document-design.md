---
title: Pourquoi M-Press Flavoured Markdown est strict
description: Pourquoi M-Press utilise un profil Markdown déterministe pour accélérer les builds et fiabiliser l'édition et la traduction.
order: 32
---

Markdown est le bon choix par défaut pour MPress. Il est familier, portable et facile
à adopter. Sa flexibilité rend cependant l'édition précise de la source plus coûteuse que
que le rendu seul ne suggère.

MPress analyse actuellement Markdown pour générer une page et réanalyse la structure de la source
pour repérer les plages sûres de traduction et d'édition rapide. La réutilisation d'un arbre de rendu transformé
change la sémantique de la source dans des cas limites. Supprimer la seconde analyse
sans remplacement affaiblirait les garanties d'édition octet par octet.

M-Press Flavoured Markdown définit un profil strict dont les conventions permettent
une analyse déterministe, des plages source stables, des composants explicites et
un échange complet avec Markdown.

## Ce que nous retenons de Djot

Djot montre qu'un format de type Markdown est plus simple à analyser lorsqu'il
supprime les règles non-locales et ambiguës. M-Press Flavoured Markdown adopte plusieurs principes :

- l'analyse doit être linéaire et sans retour en arrière;
- les références doivent ressembler à des références avant que leurs définitions soient connues;
- l'emphase ne doit pas nécessiter l'algorithme complet de flanquement de CommonMark;
- le HTML brut doit être explicite;
- une construction devrait avoir une syntaxe préférée; et
- les conteneurs arbitraires ne doivent pas exiger un langage de programmation embarqué.

M-Press Flavoured Markdown ne copie pas la grammaire de Djot. M-Press a des besoins produit différents.
Son profil doit comprendre les composants M-Press, préserver les octets Markdown importés,
résoudre les imports de projet de façon sûre, exposer des plages d'édition exactes, et prendre en charge une
implémentation mono-binaire uniquement en Go.

## Le parseur natif reste petit

Le parseur peut reconnaître un bloc à partir de la ligne courante. Il n'a pas besoin de demander
si une référence future existe, si un nom de tag HTML est valide, ou comment une
largeur d'indentation arbitraire affecte un élément de liste précédent.

Les composants placent des attributs limités à la ligne après leur nom et se ferment avec `@end`.
Ils n'entourent pas les attributs d'accolades. Les directives feuilles se terminent par un
`/`. L'indentation des listes est exacte. Les blocs HTML bruts n'exécutent pas l'analyseur inline.
Les imports créent des nœuds mais n'effectuent pas d'E/S pendant l'analyse.

Ces règles permettent un parseur d'événements en streaming. Un constructeur d'arbre, un moteur de rendu,
un extracteur de traduction ou un surligneur syntaxique peuvent consommer les mêmes événements.

## Markdown reste un format d'échange complet

Le profil ne doit pas devenir un piège à migration. M-Press traite donc
l'adaptateur Markdown comme une partie de la spécification plutôt que comme un convertisseur optionnel.

Une page Markdown importée conserve ses octets originaux et les trivia de source. Si aucun
contenu ne change, l'exportation qui préserve renvoie ces octets exactement. Si un nœud
change, l'exportateur réécrit ce nœud et conserve les plages-source inchangées.

L'export canonique crée un Markdown déterministe lorsque la préservation lexicale
Pas requis. Les composants utilisent le format brut de MPress `@component ... @end` plain extension de MPress.
L'exportateur ne produit jamais de MDX.

Les nœuds Markdown opaques sont intentionnels. Ils permettent à une extension inconnue de survivre
à un cycle d'importation et d'exportation au lieu de disparaître ou d'être devinée. Un
diagnostic rend la limitation visible.

## Les imports sont des adaptateurs de document

Un import ne colle pas du texte avant l'analyse. Il demande à un adaptateur enregistré de
produire le modèle de document commun. Markdown, OpenAPI, HTML et
des formats futurs peuvent donc entrer par des frontières séparées et testables.

Le parseur enregistre un nœud d'import sans ouvrir de fichier. La compilation le résout
plus tard, vérifie la sécurité par rapport à la racine du projet, détecte les cycles, et met en cache le résultat selon
le digest du contenu et la version de l'adaptateur.

Cette séparation empêche que le comportement réseau et du système de fichiers n'altère
les résultats du parseur. Elle permet aussi à un projet de reconstruire uniquement les documents affectés par une
source importée.

## La traduction et l'édition deviennent des sorties du parseur

Chaque nœud texte natif a une plage-source lorsqu'il est d'abord analysé. La traduction
et l'édition rapide n'ont pas besoin de redécouvrir le texte par une seconde analyse Markdown.

Le code, les destinations de lien, les attributs, les noms de composant et le HTML brut ont déjà
des types de nœuds distincts. La couche de traduction peut sélectionner les nœuds de prose et
protéger tout le reste sans reconstruire la grammaire source.

L'édition dans le navigateur peut identifier un nœud par document, plage-source, type et digest
de source. L'enregistrement vérifie toujours le digest avant d'appliquer une modification.

## Pourquoi il ne s'agit pas d'un langage de template

M-Press Flavoured Markdown n'a ni variables, ni boucles, ni expressions, ni code de module, ni JSX, ni exécution runtime
de composants à l'exécution. Ces fonctionnalités compliqueraient l'évaluation, la sécurité, la mise en cache,
et rendraient l'exportation nettement plus difficile.

Les imports composent des documents. Les composants décrivent le contenu sémantique. La configuration
contrôle le site. Le code Go implémente le rendu. Chaque préoccupation a une frontière unique.

## L'adoption doit être méritée

Le profil doit rester expérimental jusqu'à ce qu'un prototype prouve quatre points :

1. Cinq pages Wails représentatives restent agréables à rédiger.
2. Leur HTML généré correspond aux versions Markdown.
3. L'importation et l'exportation Markdown passent la suite de conformité publique.
4. L'analyse native produit une amélioration tangible de bout en bout du build.

Les conventions doivent rester agréables à écrire. Toute règle qui accélère
l'analyse mais rend Markdown plus difficile à rédiger doit rester un détail
d'implémentation interne.

Voir la [spécification M-Press Flavoured Markdown](/mpress-document/) pour la grammaire normative et
les exigences d'échange.
