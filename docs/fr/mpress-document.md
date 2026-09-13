---
title: Spécification M-Press Flavoured Markdown
description: Le profil Markdown strict utilisé par M-Press, notamment pour les métadonnées, les composants, les diagnostics et le formatage canonique.
order: 13
---

M-Press Flavoured Markdown, ou MPF, est le profil Markdown strict utilisé par
M-Press. Les fichiers source conservent l'extension `.md`. Ce document spécifie
le schéma 1.

MPF est du Markdown avec un ensemble fermé de conventions pour les métadonnées,
les composants et le traitement déterministe. Il ne s'agit pas d'un format de
document concurrent et il n'autorise ni MDX ni code exécutable.

Les mots-clés **must**, **must not**, **required**, **should**, **should not**,
, et **may** décrivent des exigences normatives.

## Objectifs

M-Press Flavoured Markdown est conçu pour:

- analyser en temps linéaire sans retour arrière;
- produire des plages de source exactes pour chaque nœud du document;
- prendre en charge l'analyse en flux et incrémentale;
- rester lisible sans rendu;
- exprimer les composants MPress sans MDX ni source exécutable;
- importer chaque construction Markdown prise en charge sans perdre de contenu;
- exporter chaque document au format Markdown sans générer de MDX;
- préserver les octets originaux du Markdown importé inchangé;
- faire en sorte que la traduction et l'édition dans le navigateur opèrent sur des plages de texte stables ; et
- signaler l'entrée malformée à la ligne où elle se produit.

MPF reste du Markdown ordinaire. Ses conventions plus strictes donnent à M-Press
un modèle de document déterministe pour les builds, les transformations, les
traductions et l'édition.

## Contrat de compatibilité Markdown

L'interopérabilité Markdown fait partie du format, et non d'un outil de commodité distinct
outil. Une implémentation MPress conforme doit fournir toutes ces garanties:

1. Tout fichier Markdown accepté par le profil sélectionné peut être importé.
2. L'importation ne supprime jamais le contenu inconnu et n'exécute jamais MDX.
3. Un fichier Markdown inchangé peut être exporté octet par octet en mode de préservation.
4. Un document modifié peut être exporté en Markdown complet et déterministe.
5. L'exportation n'exige jamais MDX, JSX, JavaScript ni un runtime de framework.
6. Les extensions non prises en charge sont conservées sous forme de Markdown opaque exact avec un diagnostic.

Le contrat couvre la préservation lexicale et le sens du document. Il ne
prétend pas que chaque dialecte Markdown tiers attribue la même signification à la
la même ponctuation. L'importateur enregistre le profil sélectionné pour que la conversion
soit reproductible.

## Conformité

Un analyseur conforme doit accepter la grammaire de cette spécification et doit
produire les types de nœud spécifiés. Un sérialiseur conforme doit émettre un
document MPress canonique.
exigences dans [Interopérabilité Markdown](#change-markdown).

Une implémentation peut imposer des limites documentées sur la taille de fichier, la longueur des lignes,
le nombre d'attributs et la profondeur d'imbrication. Elle doit signaler un diagnostic lorsqu'une limite est
dépassée. Elle ne doit pas ignorer du contenu en silence.

## Encodage et fins de ligne

Un document MPress doit utiliser UTF-8. Une marque d'ordre d'octets n'est permise qu'au
début d'un fichier et ne fait pas partie du contenu du document.

L'analyseur doit reconnaître LF, CRLF et CR comme fins de ligne. Il doit conserver les
octets de fin de ligne d'origine dans les plages source. Un sérialiseur canonique écrit LF.

Une ligne vide ne contient aucun caractère autre que des espaces ou des tabulations. Les lignes vides
séparent les constructions de bloc. Plus d'une ligne vide a la même signification
qu'une seule ligne vide, mais l'analyseur conserve chaque octet en tant que trivia source.

## Document minimal

M-Press Flavoured Markdown n'a pas de signature ou de préambule obligatoires. Il s'agit d'un
document conforme au schéma 1 :

```text
Hello world.
```

Un document sans métadonnées utilise le schéma 1. L'extension `.md` identifie
la source comme Markdown. Le champ de métadonnées `schema` sélectionne un autre
schéma M-Press lorsque cela est nécessaire.

## Métadonnées

Les métadonnées de la page utilisent un bloc délimité optionnel. L'ouverture `---` doit être la
première ligne du document et ne doit contenir aucun autre caractère. Aucune ligne vide,
d'espace ou commentaire ne peut la précéder. Une marque d'ordre d'octets UTF-8 optionnelle n'est pas
contenu du document. Une ligne de fermeture `---` doit occuper sa propre ligne.

Le corps des métadonnées est constitué de métadonnées M-Press Flavoured Markdown, pas de YAML. Chaque champ occupe
une ligne et utilise `key = value` la syntaxe.

```text
---
schema = 1
title = "Install Wails"
description = "Install the tools required to build a Wails application."
slug = "quick-start/installation"
order = 20
draft = false
tags = ["installation", "beginner"]
---
```

Une clé de métadonnées est un identifiant ASCII. Elle peut contenir des lettres, des chiffres, `_`,
`-`, et `.`, et doit commencer par une lettre ou `_`. Une valeur doit être une
chaîne JSON, un nombre, un booléen, null, un tableau ou un objet sur la même ligne logique.

Lorsqu'un bloc de métadonnées est présent, `schema` est requis, doit être son premier champ,
et doit être un entier positif. Un analyseur doit rejeter un schéma non pris en charge
avant d'analyser le contenu du document. Les clés sans préfixe suivantes sont définitivement
réservées par le schéma 1:

`schema`, `title`, `translationKey`, `description`, `draft`, `layout`, `slug`,
`order`, `date`, `template`, `banner`, `hero`, `tags`, `author`, `authors`,
`image`, `imageMode`, `imageFit`, `imageBackground`, `imageWidth`, `showTags`,
et `headingSize`.

Ces noms sont sensibles à la casse. L'ensemble réservé par le schéma 1 est figé. Un schéma ultérieur
le schéma ne doit pas attribuer une signification MPress à une autre clé sans préfixe car un
document existant peut déjà utiliser cette clé comme métadonnée personnalisée.

Toutes les clés commençant par `mpress.` sont réservées aux fonctionnalités MPress présentes et futures
fonctionnalités. Les métadonnées personnalisées peuvent utiliser toute autre clé non réservée. Les clés personnalisées avec espace de noms
telles que `wails.section`, `seo.image`, et `analytics.campaign` sont
recommandées. Les clés personnalisées inconnues sont conservées telles quelles et mises à la disposition de
l'application. Elles ne doivent jamais être réinterprétées silencieusement par un schéma ultérieur.

Les clés dupliquées sont une erreur. Une première ligne `---` ouvre toujours les métadonnées, donc un
délimiteur de fermeture manquant est une erreur. Une `---` ligne ailleurs est du
du texte en gras.

## Blocs

Un bloc commence au début d'une ligne logique. Sauf à l'intérieur d'une liste ou d'une citation,
l'indentation en début de ligne ne change pas le type de bloc. Une construction de bloc doit être
séparée des paragraphes environnants par une ligne vide.

Un analyseur doit déterminer le type de bloc à partir de la ligne courante. Il ne doit pas réinterpréter
les lignes précédentes après avoir lu les lignes suivantes.

### Paragraphes

Une ou plusieurs lignes ordinaires forment un paragraphe. Une ligne vide termine le paragraphe.
La `@end` directive pour le conteneur courant peut aussi y mettre fin. Un autre marqueur de bloc
sans ligne vide précédente reste du texte de paragraphe.

```text
MPress builds complete documentation from plain text.
This line starts on a new rendered line.

This sentence is \
continued without a rendered line break.

This is a new paragraph.
```

Une fin de ligne unique à l'intérieur d'un paragraphe crée un saut de ligne dur. Un antislash
immédiatement avant la fin de ligne supprime cette fin de ligne. Le texte sur la
ligne physique suivante continue dans le même flux en ligne. Tout espace avant le
antislash reste du contenu, donc le texte peut se renvoyer à la ligne sans fusionner les mots adjacents.

Le texte ne peut pas se transformer accidentellement en un autre bloc après le début d'un paragraphe.
Par exemple, `# text` sur la deuxième ligne d'un paragraphe reste du texte.

### Titres

Un titre commence par un à six `#` caractères suivis d'un espace ASCII.
Il doit occuper une seule ligne.

```text
# Install MPress

## Verify the installation
```

Fermeture `#` n'ont aucun sens spécial. Les titres Setext ne font pas partie de
du document MPress.

### Règles

La directive réservée `@hr` crée une séparation thématique.

```text
Before the break.

@hr

After the break.
```

### Listes

Un élément non ordonné commence par `- `. Un élément ordonné commence par un ou plusieurs
chiffres ASCII suivis de `. `. L'indentation des listes doit être un multiple de deux
espaces. Les tabulations ne sont pas autorisées pour l'indentation structurelle.

```text
- Install Go.
- Install MPress.
  - Build the documentation.
  - Run the checks.

1. Create the project.
2. Preview the site.
```

Le premier élément détermine si une liste est ordonnée. Chaque élément à cette profondeur
doit utiliser le même type de marqueur. Les numéros des éléments ordonnés sont conservés comme métadonnées source
métadonnées. Un sérialiseur canonique commence au premier numéro et incrémente
les numéros suivants.

Les lignes de continuation doivent s'aligner à deux espaces au-delà de leur marqueur d'élément. Il n'existe pas de
lignes de continuation paresseuses. Un bloc ou une liste imbriquée doit aussi commencer deux espaces
au-delà de son élément parent.

```text
- First paragraph in the item.

  Second paragraph in the same item.

  @note type="tip"
  A component in the item.
  @end
```

Un élément de tâche place `[ ]`, `[x]`, ou `[X]` immédiatement après son marqueur de liste.

```text
- [ ] Write the guide.
- [x] Run the checks.
```

### Citations

Une ligne de citation commence par `> ` ou consiste uniquement en `>`. Chaque ligne de la citation
doit porter le préfixe. Répéter le préfixe crée une citation imbriquée.

```text
> Documentation is part of the product.
>
> > Clear examples make adoption easier.
```

L'analyseur supprime un préfixe à la fois et applique la grammaire de bloc au
texte restant.

### Blocs de code

Un bloc de code utilise une fence de trois backticks ou plus. La fence d'ouverture peut être
suivie d'un identifiant de langue et d'une liste d'attributs. La fence de fermeture doit
contenir au moins autant de backticks que la fence d'ouverture et aucun autre
caractères non blancs.

`````text
````go {title="Build the site" lineNumbers=true}
site, err := mpress.Build(config)
@end
```
````
`````

Le corps est opaque. Chaque octet entre les fences d'ouverture et de fermeture est du code.
Les directives, les rôles en ligne, les liens et le formatage ne sont pas reconnus là. Un
`@`, une ligne exacte `@end` ligne, et une fence de backticks plus courte sont tous du contenu littéral.
Un auteur doit utiliser des fences d'ouverture et de fermeture plus longues lorsque le corps contient une
ligne qui fermerait autrement le bloc. Aucun `@` échappement n'est permis ni
requis dans un bloc de code.

Les attributs `language`, `title`, `filename`, `highlight`, et `lineNumbers`
ont la même signification que leurs équivalents de composant MPress Markdown.

### HTML brut

Le HTML brut est explicite.

```text
@rawHTML
<details><summary>More</summary>Content</details>
@end
```

Le contenu n'est pas analysé. C'est du HTML et la directive n'a pas d'attributs.

Le HTML brut est autorisé parce qu'il fait partie de CommonMark. JSX, les imports,
les exports, les expressions et autres constructions MDX ne sont pas exécutables et ne doivent pas
être générés par un exportateur Markdown.

### Commentaires

Les commentaires utilisent un bloc opaque et sont conservés dans l'arbre du document.

```text
@comment
Check this claim before the next release.
@end
```

Les commentaires ne sont pas rendus. Un sérialiseur Markdown les écrit comme des commentaires HTML
avec des séquences non sécurisées `--` échappées.

## Contenu en ligne

L'analyse en ligne est limitée à un paragraphe, un titre, une cellule de tableau ou une étiquette de composant
Il ne doit pas dépendre d'une définition qui apparaît plus loin dans le document.

Un délimiteur en ligne peut être échappé avec `\`. Un délimiteur non apparié est
texte. Les délimiteurs doivent être correctement imbriqués et ne doivent pas se croiser.

### Emphase et texte en gras

Les délimiteurs simples `_` créent de l'emphase. Les `*` délimiteurs doubles créent
texte.

```text
Use _emphasis_ sparingly and make *important text* clear.
```

Un délimiteur peut s'ouvrir lorsqu'il est suivi d'un octet non blanc. Il peut se fermer
lorsqu'il est précédé d'un octet non blanc. Si le délimiteur correspondant n'est pas
au sommet de la pile de délimiteurs, l'octet est du texte littéral.

Cette règle n'utilise pas les catégories Unicode, les règles d'encadrement ni les marqueurs doublés.

### Code en ligne

Une suite d'une ou plusieurs backticks ouvre un segment de code en ligne. La prochaine suite de la même
longueur le ferme. Une suite d'une longueur différente constitue le contenu. Un segment non fermé se prolonge
jusqu'à la fin de son conteneur en ligne et génère un avertissement.

```text
Run `mpress build` or use ``a ` character`` in an example.
```

Le contenu d'un segment de code est littéral. Un sérialiseur canonique choisit un délimiteur d'une
longueur d'un caractère de plus que la plus longue suite de backticks dans le contenu.

### Liens et images

Un lien en ligne utilise `[label](destination)`. Une image ajoute `!` avant le libellé.

```text
[Read the guide](/getting-started/)
![Application window](/images/application.png)
```

Le libellé peut contenir du contenu en ligne avec des crochets correctement imbriqués. Le
la destination doit figurer sur la même ligne logique. Il s'agit soit d'une séquence avec
des parenthèses équilibrées et des échappements par backslash, ou d'une chaîne entre guillemets JSON.

Les liens de référence utilisent `[label][identifier]`. La deuxième paire est requise, donc la
reconnaissance des liens est locale. Les identifiants sont sensibles à la casse.

```text
[Download Go][go-download]

@link id="go-download" destination="https://go.dev/dl/"
```

Une référence non définie reste un nœud de lien et génère un diagnostic. La syntaxe de raccourci
de référence de type `[identifier]` ne fait pas partie de M-Press Flavoured Markdown.

### Liens automatiques

Une URI absolue `http`, `https`, ou `mailto` URI entre chevrons crée un
lien automatique.

```text
<https://m-press.me>
```

Le texte brut qui ressemble à une URL reste du texte. Le lien automatique du texte nu est
une option du moteur de rendu, pas une syntaxe source.

### Raccourcis emoji

Un nom d'emoji reconnu entre deux-points crée un `emoji` nœud.

```text
Build complete :white_check_mark: Ship it :rocket:
```

Un nom d'emoji contient une ou plusieurs lettres ASCII, des chiffres, `_`, `-`, or `+`.
Les noms distinguent les majuscules et les minuscules et les noms canoniques sont en minuscules. Le schéma 1 utilise un
instantané fixe du registre de shortcodes emoji compatible GitHub publié avec
la suite de conformité. Un analyseur doit utiliser ce registre local. Il ne doit pas interroger
un service réseau ni permettre qu'une mise à jour du registre change le sens d'un
document de schéma 1 existant.

Seul un nom présent dans le registre du schéma crée un `emoji` nœud. Un nom inconnu,
y compris ses deux-points, reste du texte littéral et ne génère pas de diagnostic.
Une barre oblique inverse avant le premier deux-points empêche la reconnaissance, donc `\:rocket:` rend
comme le texte littéral `:rocket:`.

Les shortcodes emoji sont reconnus dans les titres, les paragraphes, les étiquettes de lien, les
cellules de tableau et le texte des composants. Ils ne sont pas reconnus dans les segments de code en ligne, les blocs de code,
les blocs HTML bruts, les commentaires, les valeurs de métadonnées, les valeurs d'attributs, ou
destinations. Un moteur de rendu HTML écrit la séquence Unicode native du registre.
Il ne doit pas nécessiter un service d'images, un script ou un remplacement côté client.

Canonical M-Press Flavoured Markdown écrit le nom canonique du shortcode. Un importeur Markdown
importateur utilisant le `mpress-markdown-1` le profil reconnaît le même registre.
Canonical MPress Markdown conserve le shortcode. Portable CommonMark et GFM
exportent la séquence Unicode native parce que les shortcodes emoji ne font pas partie
de ces spécifications.

### Rôles en ligne supplémentaires

Les sémantiques en ligne moins courantes utilisent un rôle explicite. Cela évite d'attribuer à la ponctuation courante
plus d'un sens.

```text
Water is H@sub[2]O.
Press @kbd[Command K].
This is @mark[important] and @delete[obsolete].
```

"La grammaire est `@name[content]`. `name` est un identifiant ASCII. Les crochets à l'intérieur
Le contenu doit être équilibré ou échappé. Les rôles définis par la version 0.1 sont
`sub`, `sup`, `mark`, `insert`, `delete`, `kbd`, `var`, et `cite`.

Un rôle inconnu reste un `inline-role` nœud. Il ne doit pas disparaître.

### Références de métadonnées

Le rôle en ligne réservé `metadata` insère une valeur de métadonnée comme texte à la
position de référence.

```text
This page is called @metadata[title].
The previous location was @metadata[wails.redirect].
```

Le contenu entre crochets est une clé de métadonnée complète et sensible à la casse.
Les points font partie de la clé, donc `wails.redirect` effectue une recherche exacte de cette
clé qualifiée par espace de noms. Elle ne traverse pas un objet. Les métadonnées réservées et personnalisées
utilisent les mêmes règles de recherche.

Une chaîne insère son contenu sans guillemets. Les nombres, les booléens et
null utilisent leur écriture JSON canonique. Les tableaux et objets utilisent une représentation JSON canonique compacte
JSON. Les clés d'objet conservent l'ordre source des métadonnées. La valeur insérée est du texte, pas
du balisage, et elle ne doit pas être analysée de nouveau pour les rôles, la mise en forme, le HTML ou les composants.
Les substitutions typographiques ne doivent pas la modifier. Un moteur HTML doit l'échapper
comme du texte ordinaire.

Une référence de métadonnée est reconnue dans tout contexte de contenu en ligne, y compris
les titres, paragraphes, libellés de lien, cellules de tableau et la prose des composants. Elle n'est pas
reconnue à l'intérieur des blocs de code, des blocs HTML bruts, des commentaires, des valeurs de métadonnées, des valeurs d'attribut,
ou des destinations de lien.

Si la clé n'existe pas, le moteur de rendu conserve le littéral
`@metadata[key]` source et produit un `mpd-metadata-reference` diagnostic. Il
ne doit pas insérer une chaîne vide ni choisir silencieusement une autre clé.

### Échappements et entités

Une barre oblique inverse avant une ponctuation ASCII rend l'octet suivant littéral. Une barre oblique inverse
précédant un autre octet reste une barre oblique inverse.

M-Press Flavoured Markdown n'interprète pas les entités HTML. Écrivez le caractère UTF-8 lui-même.
L'adaptateur Markdown conserve l'orthographe de l'entité comme provenance et la décode pour
le modèle de document.

## Composants

Un composant est soit un conteneur, soit une feuille déclarée par le registre de schéma.
Sa ligne d'ouverture commence par `@`. Le corps d'un conteneur utilise le M-Press Flavoured Markdown ordinaire,
et `@end` le ferme.

```text
@note type="tip" title="No MDX required"
Components contain *ordinary document content*.
@end
```

Les composants peuvent s'imbriquer. Le parseur utilise une pile et le plus proche `@end` ferme le
composant conteneur courant.

Le registre de schéma déclare chaque composant comme conteneur ou feuille.
Un composant feuille se compose uniquement de sa ligne d'ouverture et n'utilise pas `@end`.

```text
@image light="/light.png" dark="/dark.png" alt="Architecture"
```

MPress contrôle l'intégralité du registre des composants. Les composants définis par l'utilisateur sont
ne fait pas partie de M-Press Flavoured Markdown. Une directive inconnue produit un `mpd-directive`
diagnostic et reste du texte littéral. Il n'ouvre pas de conteneur.

### Attributs

Les attributs suivent le nom du composant sur la ligne d'ouverture. Au moins un ASCII
espace sépare le nom de son premier attribut, et un ou plusieurs espaces ASCII
séparent les attributs adjacents. Les accolades ne font pas partie de la
syntaxe des composants M-Press Flavoured Markdown.

La ligne d'ouverture utilise cette grammaire:

```text
opening    = "@" name [spaces attributes] line-end
attributes = attribute *(spaces attribute)
attribute  = name | name "=" value
value      = string | number | boolean | null | array | object
spaces     = 1*(ASCII space)
```

Les noms utilisent le metadata key grammar. Les valeurs utilisent la syntaxe JSON sur une seule ligne. Les
valeurs de chaîne non délimitées ne sont pas autorisées. Un nom sans `=` est une abréviation pour
`name=true` et n'est valide que pour un attribut booléen déclaré. Les
attributs en double sont une erreur.

L'analyse des attributs est limitée à la ligne d'ouverture. Les chaînes, tableaux et objets JSON
peuvent contenir des espaces. L'analyseur détermine leur fin d'après la syntaxe JSON,
et non en séparant la ligne à chaque espace.

Le registre de schéma détermine si une ligne d'ouverture est complète ou nécessite
une correspondance `@end`. Le parseur n'en déduit pas cela à partir de la ponctuation source. Une
barre oblique à l'intérieur d'une valeur entre guillemets, comme les barres obliques d'une URL, fait partie du contenu ordinaire
contenu. Un sérialiseur canonique écrit un espace entre le nom du composant et
chaque attribut.

Voici des lignes d'ouverture canoniques :

```text
@note type="tip" title="Read this first"
@table header search page-size=20
@image src="/images/overview.png" alt="Product overview"
@hr
```

### Lignes de directives littérales

Au début d'une ligne logique, une barre oblique inverse échappe le caractère initial d'une directive `@`.
Il s'agit d'une application de la règle normale d'échappement de la ponctuation, et non d'un mécanisme
d'échappement distinct.

Cette règle ne s'applique pas à l'intérieur des blocs de code. Le corps des blocs de code est opaque et
conservent chaque `@` sans échappement.

```text
\@note is displayed as @note.
```

### Vidéo

La vidéo est un composant feuille reposant sur l'élément vidéo natif du navigateur. Le
cas courant nécessite seulement un nom de fichier source :

```text
@video src="setup.mp4"
```

MPress ajoute des contrôles natifs, la lecture intégrée, le préchargement des métadonnées, une
mise en page réactive, une taille intrinsèque de 1280 par 720, le type MIME inféré, et un lien de secours
lien. Les auteurs ne répètent pas ces valeurs par défaut.

Le même `src` l'attribut accepte un tableau lorsqu'une vidéo comporte plusieurs encodages.
Le navigateur les essaie dans l'ordre des sources. `base` préfixe chaque chemin média relatif
chemin, y compris les sources, le poster, les sous-titres et la transcription:

```text
@video base="/media/setup" src=["setup.webm","setup.mp4"] poster="poster.webp" captions=["captions.en.vtt","captions.fr.vtt"] title="Setup walkthrough" transcript="transcript.html"
```

Les attributs optionnels sont délibérément limités:

- `base` supprime un répertoire répété des chemins média relatifs.
- `poster` définit l'image affichée avant la lecture.
- `captions` accepte un nom de fichier WebVTT ou un tableau. La première piste est la
  piste par défaut.
- `lang` fournit la langue des sous-titres lorsqu'elle ne peut pas être déduite. Par défaut
  à `en`.
- `title` fournit le nom accessible et le sous-titre visible.
- `transcript` ajoute un lien vers la transcription.
- `width` et `height` remplacent les dimensions intrinsèques 1280 par 720. Le CSS maintient
  le lecteur adaptatif.

Les noms de fichiers de sous-titres peuvent inclure une étiquette de langue BCP 47 immédiatement avant le
extension, par exemple `captions.en.vtt` ou `captions.pt-BR.vtt`. MPress déduit que
l'étiquette avant d'utiliser `lang`. Les types MIME vidéo sont déduits de `.mp4`, `.m4v`,
`.webm`, `.ogv`, `.ogg`, et `.mov` noms de fichiers. Les chemins relatifs et les URL HTTP ou HTTPS
Les URL sont acceptées. Les schémas d'URL non sûrs sont rejetés.

MPress n'ajoute pas autoplay, des contrôles de lecture personnalisés, ni un lecteur JavaScript
API. Un site qui exige une politique de lecteur non standard peut utiliser `@rawHTML`; ces cas
n'agrandissent pas le composant vidéo portable.

## Tableaux

Un tableau est un bloc explicite. Chaque ligne commence et se termine par `|`. Un antislash
échappe une barre verticale littérale.

```text
@table header=true
| Name | Purpose |
| Markdown | Portable authoring |
| M-Press Flavoured Markdown | Deterministic authoring |
@end
```

Chaque ligne doit contenir le même nombre de cellules. Les espaces ASCII en début et en fin
dans une cellule sont des éléments de formatage et ne font pas partie du contenu de la cellule. Les cellules utilisent
la grammaire en ligne.

L'alignement des colonnes utilise un `align` tableau avec `left`, `centre`, `right`,
ou `default` valeurs.

```text
@table header=true align=["left", "right"]
| Stage | Time |
| Parse | 83 ms |
@end
```

Les tableaux peuvent offrir des contrôles pour le lecteur. Chaque contrôle est facultatif et désactivé par
défaut:

- `search` ajoute un champ de recherche insensible à la casse pour toutes les cellules.
- `filter` ajoute un menu de filtre par valeur exacte à chaque en-tête de colonne.
- `sort` rend chaque en-tête triable par ordre croissant ou décroissant.
- `paginate` divise les lignes correspondantes en pages. `page-size` définit le nombre de
  lignes par page et a pour valeur par défaut `10`.
- `column-separators` trace la bordure de chaque colonne dans l'en-tête et le corps.

```text
@table header search filter sort paginate column-separators page-size=10
| Package | Platform | Downloads |
| Core | Linux | 1840 |
| Team | macOS | 920 |
| Core | Windows | 1260 |
@end
```

La recherche, les filtres, le tri et la pagination se combinent. Un changement de recherche ou de filtre
ramène à la première page. Le tri est stable, sensible aux paramètres régionaux, et compare
les nombres intégrés numériquement. Les contrôles générés doivent avoir des noms accessibles,
les en-têtes triables doivent exposer `aria-sort`, et le nombre de lignes visibles doit être un
`aria-live` statut. Sans support de script, chaque ligne reste visible et le
la table sous-jacente reste lisible.

## Notes de bas de page

Une référence de note de bas de page en ligne utilise `[^identifier]`Sa définition est un
bloc explicite et peut contenir n'importe quel contenu de bloc.

```text
The result is reproducible.[^benchmark]

@footnote id="benchmark"
The benchmark uses twenty paired cold builds.
@end
```

Les identifiants sont sensibles à la casse. Une définition en double est une erreur. Une
référence non définie génère un diagnostic sans supprimer la référence.

## Imports and includes

Un include lit un autre fichier ou fragment M-Press Flavoured Markdown. Un import invoque un
format adapter.

```text
@include src="shared/prerequisites.md"

@import src="README.md" format="markdown" profile="gfm"

@import src="api/openapi.yaml" format="openapi"
```

`src` est résolu par rapport au fichier contenant. La résolution doit rester
à l'intérieur du répertoire racine du projet configuré sauf si le projet autorise explicitement un
répertoire racine supplémentaire. Les liens symboliques doivent être résolus avant cette vérification.

Les imports distants sont désactivés par défaut. Un adaptateur distant doit fixer le contenu immuable par digest
et ne doit pas récupérer le contenu pendant une compilation hors ligne.

Le parseur crée un nœud d'import sans ouvrir la cible. La résolution d'import
est une phase de construction distincte. Cela rend l'analyse déterministe et permet la mise en cache.

Un adaptateur renvoie le même modèle de nœud de document que le parseur natif. Chaque
nœud importé enregistre son format source, le fichier source, la plage d'octets quand disponible,
et la version de l'adaptateur.

Un cycle d'inclusion est une erreur. Les implémentations doivent signaler le cycle complet.

### Modes d'importation

L'optionnel `mode` attribute a ces valeurs :

| Mode | Résultat |
| --- | --- |
| `fragment` | Insérer les nœuds de bloc importés à la directive. C'est le comportement par défaut. |
| `document` | Importer les métadonnées et le corps. Les conflits de métadonnées sont des erreurs sauf s'ils sont mappés explicitement. |
| `code` | Importer les octets comme un bloc de code. |
| `data` | Rendez les données structurées disponibles pour un composant enregistré sans ajouter de nœuds au corps. |

## Échange Markdown

Markdown est un adaptateur requis. MDX ne l'est pas.

La version 0.1 définit ces profils:

- `commonmark-0.31.2`;
- `gfm` pour les tableaux, les listes de tâches, le texte barré, et les liens automatiques ; et
- `mpress-markdown-1` pour la grammaire complète du composant Markdown de MPress,
  les notes de bas de page, les substitutions typographiques, et le frontmatter de page.

Le profil par défaut est `mpress-markdown-1`.

### Garantie d'importation

L'adaptateur doit conserver chaque octet d'un fichier source Markdown. Il ne doit jamais
rejeter une extension inconnue ou une construction non prise en charge.

Les constructions connues deviennent des nœuds de document typés. Le HTML brut devient un nœud HTML brut.
La syntaxe de type MDX devient un nœud Markdown opaque et un diagnostic. Elle n'est jamais
exécutée.

Si une construction est du Markdown valide mais n'a pas d'équivalent natif dans le Document MPress, l'
adaptateur crée un `opaque-markdown` nœud contenant ses octets sources exacts.
Il s'agit de la solution de repli sans perte, pas d'un raccourci de récupération d'erreur.

Chaque nœud importé enregistre :

- la plage d'octets d'origine ;
- les trivia en début et en fin ;
- l'orthographe du délimiteur d'origine ;
- les terminaisons de ligne source ;
- le profil Markdown ; et
- si le nœud ou l'un de ses enfants a été modifié.

### Garantie d'exportation

Un exportateur Markdown a trois modes.

| Mode | Garantie |
| --- | --- |
| `preserve` | Un document Markdown importé non modifié est identique octet par octet. Les nœuds modifiés sont insérés dans les plages originales non touchées. |
| `canonical` | Émettez du Markdown déterministe représentant le modèle de document complet. Le formatage peut changer, mais le contenu et la structure ne doivent pas. |
| `portable` | Émettez CommonMark ou GFM sans composants MPress quand une représentation portable existe. Signalez les composants qui ne peuvent pas être réduits sans perte. |

L'exporteur doit générer du Markdown, pas du MDX. Il ne doit pas émettre d'importations, d'exportations,
d'éléments JSX, d'expressions JavaScript ou d'appels à des composants de framework.

Les composants MPress s'exportent avec le MPress `@component ... @end` Markdown
extension. Un lecteur Markdown générique voit donc le texte en clair et le composant
corps plutôt que la syntaxe exécutable.

Le Markdown MPress canonique conserve `@metadata[key]`. Le Markdown portable résout
la référence et écrit sa valeur en texte clair. Preserve mode conserve le
octets originaux de la source.

### Cartographie Markdown

| élément Markdown | Représentation du document MPress |
| --- | --- |
| Paragraphe | Paragraphe |
| Titre ATX ou Setext | Sur une seule ligne `#` titre |
| Emphase | `_text_` |
| Emphase forte | `*text*` |
| Code en ligne | Portion de code |
| Code délimité ou indenté | Délimiteur de code Backtick |
| Bloc de citation | Lignes de citation préfixées |
| Liste ordonnée ou non ordonnée | Liste stricte |
| Liste de tâches | Liste stricte avec état des tâches |
| Lien ou image | Nœud de lien ou d'image |
| Lien de référence | Lien de référence et `@link` définition |
| Shortcode d'emoji | `:emoji_name:` |
| Séparateur thématique | `@hr` |
| Tableau GFM | `@table` |
| Note de bas de page | `@footnote` |
| HTML brut | `@rawHTML` |
| Composant MPress | Composant ou composant feuille |
| Référence de métadonnées | `@metadata[key]` |
| Frontmatter YAML | Métadonnées M-Press délimitées avec les valeurs converties sans perte |
| Extension non prise en charge | nœud `opaque-markdown` exact |

### Provenance persistante

La provenance fait directement référence au fichier source `.md`. L'exportation
Markdown canonique ne nécessite aucun fichier supplémentaire.

Une exportation en mode `preserve` conserve les octets source, leur empreinte,
le profil Markdown et les plages de nœuds dans le modèle de compilation. Un
exportateur doit vérifier l'empreinte avant de réutiliser ces octets. Si elle ne
correspond pas, l'exportateur doit utiliser le mode canonique ou s'arrêter avec
une erreur. Il ne doit pas prétendre préserver les octets.

## Modèle de document

Le modèle normatif est un arbre ordonné de nœuds. Chaque nœud contient :

- `kind`;
- `source` avec le fichier, l'octet de début, l'octet de fin, la ligne de début et la colonne de début ;
- attributs ordonnés ;
- enfants ordonnés lorsque le type les permet ;
- provenance de la source lors de l'importation ; et
- diagnostics rattachés à ce nœud.

Le contenu textuel renvoie, autant que possible, à des tranches de source immuables. Un analyseur n'est pas
tenu d'allouer une chaîne pour chaque nœud de texte.

Les types principaux de blocs sont `document`, `metadata`, `paragraph`, `heading`, `rule`,
`list`, `item`, `quote`, `code`, `raw-html`, `comment`, `table`, `table-row`,
`table-cell`, `footnote`, `component`, `import`, `include`, et
`opaque-markdown`.

Les types principaux en ligne sont `text`, `soft-break`, `hard-break`, `emphasis`,
`strong`, `code-span`, `link`, `image`, `automatic-link`, `emoji`,
`footnote-reference`, `metadata-reference`, et `inline-role`.

## Exigences du parseur

Un analyseur natif conforme doit :

1. parcourir l'entrée du début à la fin sans revenir en arrière ;
2. utiliser un regard en avant limité à la ligne courante ;
3. reconnaître les blocs avant d'analyser leur contenu en ligne ;
4. protéger les délimitations de code opaques avant de résoudre les composants ;
5. résoudre les composants avec une pile ;
6. résoudre les délimiteurs en ligne avec une pile ;
7. analyser les références sans consulter les définitions ultérieures ;
8. maintenir la résolution des imports en dehors du parseur ;
9. récupérer à une ligne vide ou du conteneur courant `@end`; et
10. émettre des plages source même lorsqu'un nœud contient un diagnostic.

Pour `n` octets d'entrée et profondeur d'imbrication `d`, l'analyse native doit être `O(n)` en temps et
`O(d + r)` mémoire auxiliaire, où `r` est le nombre de références conservées par
le document. L'arborescence du document elle-même n'est pas une mémoire auxiliaire.

Un analyseur conforme ne doit pas nécessiter d'expressions régulières, de tokenisation HTML,
de recherche de catégorie Unicode, de résolution d'entités, d'accès réseau ou d'analyse spécifique au composant
pour découvrir la structure du document.

## Diagnostics et récupération

Un diagnostic contient un code stable, une gravité, un message, une plage source et une
suggestion de réparation optionnelle.

Les codes de diagnostic requis incluent :

| Code | Signification |
| --- | --- |
| `mpd-version` | Schéma de document non pris en charge |
| `mpd-metadata` | Métadonnées invalides, dupliquées ou non terminées |
| `mpd-metadata-reference` | Référence de métadonnées non définie |
| `mpd-utf8` | UTF-8 invalide |
| `mpd-unclosed` | Composant non fermé, clôture de code ou bloc HTML brut non fermé |
| `mpd-end` | Inattendu `@end` |
| `mpd-attribute` | Attribut invalide ou dupliqué |
| `mpd-directive` | Directive inconnue ou indisponible |
| `mpd-indent` | Indentation structurelle invalide |
| `mpd-reference` | Référence indéfinie ou dupliquée |
| `mpd-import-cycle` | Cycle d'inclusion ou d'importation |
| `mpd-import-root` | L'importation dépasse une racine autorisée |
| `mpd-opaque` | Construct Markdown conservé comme contenu opaque |

Un élément inattendu `@end` devient texte littéral après avoir produit un diagnostic. Un
composant ordinaire non fermé se ferme à la fin de son bloc contenant. Un
bloc HTML brut non fermé consomme le fichier restant et génère un diagnostic. Aucune
règle de récupération ne peut supprimer des octets source.

## Sérialisation canonique

Document MPress canonique :

- utilise des fins de ligne LF;
- omet les métadonnées lorsque le document n'en contient pas et utilise le schéma 1;
- écrit `---` sur la première ligne lorsque des métadonnées sont présentes;
- écrit `schema` comme le premier champ de métadonnées;
- écrit les métadonnées dans l'ordre source;
- utilise une ligne vide entre les blocs frères;
- utilise deux espaces par niveau de liste;
- utilise l'échappement JSON pour les valeurs d'attribut;
- écrit les attributs dans l'ordre source;
- utilise une clôture de code d'un backtick de plus que la clôture de fermeture potentielle la plus longue
  dans le corps, avec une longueur minimale de trois;
- utilise les valeurs booléennes et null en minuscules;
- écrit des lignes explicites `@end` pour les composants conteneurs;
- omet `@end` pour les composants feuilles; et
- se termine par un seul saut de ligne.

La sortie canonique doit être analysée en un modèle de document équivalent au modèle d'entrée.

## Tests de conformité

La suite de conformité publique doit inclure:

- un exemple pour chaque règle de syntaxe normative;
- des entrées malformées et adversariales pour chaque règle de récupération;
- des composants profondément imbriqués jusqu'à la limite documentée;
- importations Markdown mixtes LF, CRLF et CR ;
- tous les exemples CommonMark 0.31.2 ;
- les exemples applicables de GitHub Flavoured Markdown ;
- chaque fixture de composant MPress Markdown ;
- import Markdown suivi d'un export préservé avec égalité d'octets ;
- import Markdown suivi d'un export canonique et d'une réimportation sémantique ;
- export d'un M-Press Flavoured Markdown vers Markdown suivi d'une réimportation sémantique ;
- tests de fuzz affirmant une progression linéaire et l'absence de panics ; et
- benchmarks sur corpus utilisant la documentation complète de Wails v3.

L'implémentation Go initiale devrait satisfaire ces objectifs de performance non normatifs
cibles sur le corpus Wails v3:

- au moins deux fois le débit d'analyse de Goldmark pour un contenu natif équivalent ;
- pas plus de la moitié des allocations du parseur de Goldmark ;
- extraction des plages source pendant l'analyse native, sans seconde analyse ; et
- HTML généré inchangé comparé aux fixtures MPress Markdown équivalentes.

Les revendications de performance doivent indiquer la révision du corpus, la version de Go, le nombre de CPU,
le nombre d'échantillons, les distributions avant et après, et la comparaison des sorties générées.

## Gestion des versions

Les métadonnées `schema` le numéro change lorsque la source valide existante peut acquérir un
sens différent du document ou lorsque le registre des composants change. Un document
sans métadonnées est le schéma 1. Les types de nœuds additionnels, les rôles, les adaptateurs et
les attributs qui ne modifient pas la classification des directives ne nécessitent pas un nouveau
schéma lorsqu'un analyseur plus ancien peut les conserver comme nœuds inconnus.

Le délimiteur de métadonnées de la première ligne et le `schema` grammaire des champs sont stables
à travers les versions de schéma. Cela permet à un analyseur de sélectionner le schéma avant qu'il
analyse le corps du document. Un futur schéma doit utiliser le `mpress.*` espace de noms pour
nouvelles métadonnées appartenant au moteur plutôt que de revendiquer une clé personnalisée existante.

Pendant la version 0.x, ce document peut changer après des expérimentations d'implémentation.
M-Press ne doit pas décrire un schéma comme stable tant que sa grammaire,
l'adaptateur Markdown et la suite de conformité n'ont pas été livrés ensemble.
