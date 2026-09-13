---
title: Contribuer depuis une page publiée
description: Modifiez une page publiée dans le navigateur, puis transférez le brouillon vers une copie de travail locale sécurisée.
order: 35
---

Les propriétaires du site peuvent permettre aux lecteurs de démarrer une contribution locale depuis n'importe quelle page publiée.
Le lecteur n'a pas besoin de chercher le dépôt source ni le fichier Markdown correspondant
fichier.

## Faire une modification rapide

1. Sélectionnez l'icône d'édition dans la barre de navigation de la documentation.
2. Sélectionnez **Modifier rapidement cette page**.
3. Sélectionnez une phrase ou un titre surligné.
4. Saisissez le texte corrigé.

M-Press enregistre chaque modification dans le stockage local du navigateur. La barre de modification rapide
indique le nombre de modifications enregistrées. Le brouillon reste disponible si vous actualisez ou
réouvrez la page dans le même navigateur.

Sélectionnez **Ajouter un paragraphe** pour insérer un paragraphe après le bloc de texte actif.
Sélectionnez **Supprimer** pour supprimer le brouillon complet et restaurer la page publiée.

## Formater le texte

Sélectionnez du texte dans un bloc modifiable. La barre d'outils de formatage suit la sélection.
Utilisez-la pour appliquer :

- **Gras** écrit `**bold text**`.
- **Italique** écrit `_italic text_`.
- **Code en ligne** écrit une plage de code sécurisée délimitée par des backticks.
- **Ajouter un lien** demande l'adresse du lien et écrit `[label](address)`.

Vous pouvez aussi appuyer sur `Ctrl+B`, `Ctrl+I`, ou `Ctrl+K`. Sur macOS, utilisez `Command`
au lieu de `Ctrl`.

La page affiche le formatage sélectionné pendant l'édition. Le brouillon du navigateur stocke
Markdown, pas du HTML généré. Le contenu collé est inséré en texte brut. Les liens
peuvent utiliser HTTP, HTTPS, une adresse e-mail, un numéro de téléphone, des ancres de page ou des chemins relatifs.

@note{type="info" title="A browser draft does not change the site"}
Le brouillon existe uniquement dans votre navigateur. Les autres lecteurs continuent de voir la
page publiée.
@end

## Continuer sur votre ordinateur

Quand le brouillon est prêt :

1. Sélectionnez **Continuer sur l'ordinateur** dans la barre de modification rapide.
2. Conservez le `.mpress-draft` fichier téléchargé dans votre dossier Téléchargements.
3. Copiez la commande sélectionnée pour votre système d'exploitation.
4. Exécutez la commande dans un terminal.
5. Suivez le guide de contribution dans le site M-Press local.

Sans brouillon navigateur, sélectionnez **Travailler sur votre ordinateur** dans la fenêtre de contribution
M-Press ouvre toujours la même page en local.

La commande générée identifie la page en cours. Elle utilise cette forme lorsque
M-Press est déjà installé :

```sh
mpress contribute https://docs.example.com/guide/install/
```

Le script shell ou PowerShell spécifique au site télécharge M-Press lorsqu'il n'est pas
installé. La commande copiée contient le nom du fichier téléchargé, pas le contenu du brouillon
du fichier. N'éditez pas le fichier du brouillon.

Si M-Press est déjà installé, la commande équivalente est :

```sh
mpress contribute https://docs.example.com/guide/install/ \
  --draft-file mpress-draft-a1b2c3d4e5f6.mpress-draft
```

M-Press recherche un nom de fichier dans le répertoire courant et dans le répertoire
de Téléchargements configuré. Indiquez le chemin complet si votre navigateur l'a
enregistré ailleurs.

M-Press fait ensuite :

1. Trouve le dépôt, la branche source, le fichier Markdown et la route de la page.
2. Clone le dépôt ou réutilise sa copie de travail de contribution sécurisée.
3. Crée une branche nommée `contribute/<date>-<time>`.
4. Applique le brouillon du navigateur au fichier Markdown correspondant.
5. Démarre le serveur de développement et ouvre la même page.

L'assistant de contribution local peut aussi ouvrir des outils de traduction, les paramètres du site,
ou des vérifications du projet. Le rechargement en direct affiche les modifications Markdown enregistrées dans le navigateur.

## Comment fonctionnent les brouillons du navigateur

M-Press utilise les mêmes segments de texte pour la modification rapide et la traduction. Lors d'une
construction, M-Press attribue à chaque segment de texte visible un ID stable, un hachage source et
une plage d'octets exacte dans le fichier Markdown.

Le navigateur ne stocke que la source de la page, la révision source, la route et les
segments modifiés. Il ne stocke pas de copie HTML générée de la page. Cela empêche une
conversion HTML-vers-Markdown de supprimer la syntaxe Markdown.

Quand vous continuez sur votre ordinateur, le navigateur télécharge ces informations sous la forme d'un
fichier JSON au nom unique `.mpress-draft` M-Press lit ce fichier localement. Il
n'envoie pas le brouillon à un service M-Press, et le contenu du brouillon n'apparaît pas
dans l'historique de votre shell. Le fichier contient néanmoins vos modifications en texte
clair, supprimez-le donc lorsque vous n'en avez plus besoin.

Avant d'écrire la modification, M-Press vérifie que :

- le brouillon appartient à la page source sélectionnée ;
- le chemin source reste à l'intérieur du projet extrait ;
- la révision source correspond toujours à la page publiée ;
- chaque plage de texte originale contient toujours le Markdown attendu ; et
- les modifications ne se chevauchent pas.

Si l'une de ces vérifications échoue, M-Press s'arrête sans modifier le fichier. Démarrez un
nouveau brouillon depuis la page publiée actuelle ou modifiez le Markdown localement.

## Portée actuelle de la modification rapide

La modification rapide est destinée aux petites corrections de documentation. Elle prend en charge :

- texte de paragraphe ordinaire ;
- titres Markdown ;
- gras, italique, code en ligne et nouveaux liens ; et
- un nouveau paragraphe après un bloc modifiable.

M-Press protège les liens, les blocs de code, les tableaux, le HTML brut et la structure des composants
contre l'édition directe. Sélectionnez **Travailler sur votre ordinateur** lorsqu'un changement nécessite ces
fonctionnalités ou modifie la structure de la page.

## Sécurité et authentification

La page publiée contient l'URL du dépôt, la branche source, la route de la page,
le chemin source et les informations de segment de modification rapide. Elle ne contient pas
d'identifiants. Le texte des segments est déjà visible sur la page publiée.

Les dépôts publics se clonent sans authentification. Les dépôts privés utilisent
l'aide-mémoire d'identifiants Git existant du lecteur ou la connexion GitHub CLI. M-Press refuse de
réutiliser un répertoire non lié. Il refuse aussi une copie de travail existante modifiée sauf si
cette copie est déjà sur une branche de contribution M-Press.

Rien n'est publié automatiquement. La contribution reste dans la copie de travail locale
jusqu'à ce que le lecteur décide de la valider et de la soumettre.

## Activer l'action de contribution

Ajoutez la section de contribution à `mpress.yaml`:

```yaml
contribution:
  enabled: true
  repository: https://github.com/example/docs.git
  branch: main
```

Vous pouvez aussi l'activer en mode développement. Ouvrez **Modifier la configuration**, sélectionnez
**Liens**, et activez **Permettre aux lecteurs de contribuer**.

La compilation de production inclut des scripts de contribution spécifiques au site dans
`/mpress-contribute.sh` et `/mpress-contribute.ps1`Chaque script contient le
dépôt et la branche configurés. La commande dans la barre de navigation télécharge et exécute le
script correspondant depuis le site que le lecteur consulte. Il transmet également le
URL de la page en cours et le nom de fichier facultatif du brouillon téléchargé.

Les deux scripts utilisent `mpress` directement lorsqu'il est déjà sur le PATH. Sinon,
le script POSIX prend en charge Linux et macOS sur AMD64 et ARM64. Il tente `curl`
puis `wget`, télécharge l'archive correspondante depuis la dernière release GitHub,
et la vérifie par rapport à `checksums.txt` avant de l'exécuter. Le script Windows
utilise PowerShell pour télécharger le ZIP correspondant et applique le même contrôle SHA-256
de vérification. Les deux scripts exécutent `mpress contribute <page-url>` et suppriment les
fichiers temporaires lorsque la commande se termine.
