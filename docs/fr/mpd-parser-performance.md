# Performances du parseur M-Press Flavoured Markdown

Ce rapport consigne la première implémentation native du parseur M-Press Flavoured Markdown et
les travaux d'optimisation achevés le 11 août 2026.

## Système de test

- Révision avant le travail sur le parseur : `859b183`
- Go : `go1.26.5 linux/amd64`
- CPU : AMD Ryzen 9 3950X, 16 cœurs et 32 processeurs logiques
- Noyau : Linux 7.1.3 x86-64
- Corpus de bench : 10,000 pages Markdown déterministes et 256 SVG référencés
  ressources
- Source Markdown : 8,694,682 octets
- Données des ressources et des inclusions partagées : 118,156 octets
- SHA-256 du corpus : `98e9e0d411825058c03395b49ac7f18e0fb478e6379a6c193a37773f610a6ba3`

Le corpus contient dix modèles de page. Ensemble, ils exercent les métadonnées, le texte,
la syntaxe inline, le code, les listes, les tâches, les citations imbriquées, les tableaux, les liens de référence,
les notes de bas de page, les imports, les ressources d'image et les composants imbriqués.

Générez et vérifiez-le avec :

```sh
go run ./cmd/mpd-corpus -output /tmp/mpd-corpus -pages 10000 -assets 256
```

La commande relit chaque fichier généré, recalcule la somme de contrôle du manifeste,
analyse toutes les 10,000 pages, rejette les erreurs du parseur et confirme chaque
ressource.

## État initial

Le parseur initial utilisait déjà un arbre d'arène et des tranches de source immuables, mais
calculait la position de chaque nœud en balayant depuis l'octet zéro. Dix échantillons d'une itération
ont pris environ 628 à 726 ms par corpus, avec une médiane de 693 ms.
Il a alloué environ 93,194,000 octets et 260,000 objets par corpus.

Le premier profil CPU a attribué 79.7 % des échantillons à `positionAt`. Cela a montré
que les balayages répétés des positions rendaient l'implémentation effectivement quadratique pour
les pages riches en nœuds.

## Historique des optimisations A/B

Chaque changement accepté a été mesuré sur le corpus complet en mémoire de 10,000 pages
Les changements qui n'amélioraient pas la distribution ont été annulés.

| Expérience | Résultat | Décision |
| --- | ---: | --- |
| Construire les débuts de ligne une seule fois au lieu de rescanner depuis l'octet zéro | Environ 74 % plus rapide | Accepté |
| Préallouer les arènes de nœuds et d'attributs à partir de la taille de la source | Médiane d'environ 180 ms à 165 ms ; 24.5 % d'octets alloués en moins | Accepté |
| Combiner la décision ASCII avec l'indexation de la source | Médiane d'environ 165 ms à 155 ms | Accepté, remplacé plus tard par des positions paresseuses |
| Scanner JSON sans allocation | Environ 12 % plus rapide | Accepté |
| Basculements de composants à schéma fermé et vérifications des doublons sur les plages source | Médiane d'environ 137 ms à 121 ms ; allocations 176,000 à 47,000 | Accepté |
| Construire l'index de lignes uniquement pour un diagnostic ; suivre les positions inline vers l'avant | Médiane d'environ 121 ms à 106 ms ; environ 33,000 allocations | Accepté |
| Balayage ASCII 64 bits | Environ 5 % plus rapide | Accepté |
| Sauter directement aux octets délimiteurs inline | Environ 7 % plus rapide | Accepté |
| Recherche de lignes vectorisée avec recherche de CR bornée | Plusieurs pourcents plus rapide et restaure un balayage strictement linéaire | Accepté |
| Dimensionner l'arène d'attributs à partir de la longueur de la source | Atteint trois allocations par page valide | Accepté |
| Dimensionner l'arène de nœuds à partir de la longueur de la source | Environ 3.5 % plus rapide et 24.8 % de mémoire en moins | Accepté |
| Mettre en cache la ligne courante après la vectorisation de la recherche de lignes | Environ 2 % plus rapide | Accepté |
| Recherche statique du délimiteur inline au lieu de `bytes.IndexAny` | Environ 4.3 % plus rapide | Accepté |
| `binary.LittleEndian.Uint64` Mots ASCII | Environ 4.1 % plus rapide | Accepté |
| Stocker les diagnostics uniquement dans l'arène du document | Taille du nœud 64 à 56 octets ; environ 9 % de mémoire de nœud en moins | Accepté |
| Tampon de référence inline à quatre enregistrements | A récupéré presque tout le coût de validation des références sans affaiblir la gestion des débordements | Accepté |
| Détecter une fois les documents avec uniquement LF et ignorer les recherches de CR par ligne | Environ 4.3 % plus rapide | Accepté |
| Dérouler la recherche fixe du délimiteur inline huit octets à la fois | Environ 2.4 % plus rapide | Accepté |
| Mettre en cache une ligne avec le scanner scalaire antérieur | Environ 1.2 % plus lent | Rejeté |
| Remplacer la recherche de lignes par `bytes.IndexAny` | Environ 4.8 % plus lent | Rejeté |
| Ajouter des hachages d'attributs de 16 bits ou 32 bits | Médiane 79.6 ms contre 76.9 ms pour des plages exactes | Rejeté |
| Recherche statique d'octets spéciaux JSON | Neutre à légèrement plus lent | Rejeté |
| Combiner la détection ASCII et CR dans une seule boucle sur des mots Go | Environ 5.5 % plus lent que la recherche d'octets de la bibliothèque standard | Rejeté |
| Propager le drapeau de chemin rapide CR aux traceurs de position inline | Environ 2% plus lent | Rejeter |
| Ajouter un filtre de collision d'attributs dupliqués de 64 bits | Environ 1% plus lent | Rejeter |
| Utiliser zéro au lieu de `0xffffffff` pour les sentinelles de lien d'arbre | Environ 1.8% plus lent | Rejeter |
| Dérouler le balayage des chaînes JSON | Neutre à environ 0.6% plus lent | Rejeter |
| Utiliser des recherches en assembleur pour les guillemets et les échappements des chaînes JSON | Neutre à environ 0.4% plus lent | Rejeter |

Le profil final n'est plus dominé par un algorithme évitable. Ses plus grands
coûts plats appartenant au parseur : la construction des nœuds d'arène à 9.01%, l'analyse inline à
5.92%, la validation des chaînes JSON à 5.15%, la classification ASCII à 3.22%, et
la recherche de délimiteurs à 2.70%. Chacun de ces domaines a soit un gain A/B conservé soit
une alternative plus rapide mais rejetée enregistrée ci-dessus. L'espace d'allocation est le
arbre de document renvoyé : 98.42% de l'espace d'allocation mesuré se trouve dans
`ParseWithOptions`. D'autres réductions testées ont soit déplacé le travail ailleurs soit
ont ralenti l'ensemble du corpus.

## Comparaison finale

Exécutez cette comparaison avec :

```sh
go test ./internal/mpd -run '^$' \
  -bench '^(BenchmarkParseCorpus10000|BenchmarkGoldmarkCorpus10000)$' \
  -benchtime=10x -count=10 -benchmem
```

Dix échantillons finaux ont produit ces distributions :

| Parseur | Minimum | Médiane | Maximum | Octets médians | Allocations médianes |
| --- | ---: | ---: | ---: | ---: | ---: |
| Parseur M-Press natif | 58.62 ms | 64.47 ms | 67.02 ms | 47,007,648 | 30,000 |
| Goldmark | 288.55 ms | 324.66 ms | 355.09 ms | 153,346,214 | 1,154,132 |

Sur ce corpus, le parseur natif est 5.04 fois plus rapide que Goldmark. Il utilise
30.65% des octets alloués par Goldmark et 2.60% de son nombre d'allocations. Chaque
page valide nécessite exactement trois allocations : le document, l'arène de nœuds,
et l'arène d'attributs.

La médiane finale du natif est environ 90.7% inférieure à la médiane initiale de 693 ms.
Cette comparaison est délibérément prudente car Goldmark analyse un langage moins
structuré et n'extrait pas le modèle source complet.

Avec `GOMAXPROCS=1`, huit échantillons ont varié de 38.54 à 40.87 ms avec une médiane
de 38.92 ms. Le parseur est mono-thread et n'a pas de régression sur un seul CPU. La
baisse de temps provient d'une réduction de la surcharge du ramasse-miettes concurrent pendant ce
benchmark lourd en allocations.

## Critères de validité

L'implémentation optimisée a passé tous ces critères :

- chaque fixture de syntaxe enregistrée s'analyse sans diagnostic d'erreur;
- la suite golden HTML de production reste identique octet pour octet;
- toutes les 10,000 pages générées sur disque s'analysent et tous les 256 assets se vérifient;
- LF, CRLF, CR, UTF-8, métadonnées, limites, imbrication de composants, opacité du code,
  tables, listes, références, notes de bas de page et récupération ont des tests dédiés;
- les identifiants de références JSON échappés et les comptes de références au-delà du tampon inline
  ont des tests de régression;
- les délimiteurs non appariés conservent un arbre syntaxique entièrement connecté;
- les tests de concurrence passent pour les paquets parser, corpus et conformance;
- un fuzzing étendu du parseur a trouvé et isolé deux régressions de non-progrès liées à l'indentation en tête,
  régressions, puis la dernière exécution propre a complété 270,299 exécutions sans
  panic ni violation d'invariant ; et
- le run de fuzz différentiel final du scanner JSON a complété 371,797 exécutions contre
  `encoding/json.Valid` sans discordance.

Les commandes du profil final sont :

```sh
go test ./internal/mpd -run '^$' -bench '^BenchmarkParseCorpus10000$' \
  -benchtime=100x -cpuprofile=/tmp/mpd.cpu.pprof \
  -memprofile=/tmp/mpd.heap.pprof -benchmem
go tool pprof -top /tmp/mpd.cpu.pprof
go tool pprof -top -alloc_space /tmp/mpd.heap.pprof
```
