---
title: Gestion des versions
description: Capturer, vérifier et monter des versions immuables de la documentation.
order: 9
---

Les versions M-Press sont des builds statiques complets, pas des arbres sources alternatifs. Une
version capturée reste déployable même lorsque la documentation actuelle change.
Le workflow complet de versions et le sélecteur de versions sont intégrés.

## Ce qui est inclus

M-Press peut:

- capturer une build statique complète sous une étiquette de publication;
- écrire un manifeste SHA-256 pour chaque fichier capturé;
- vérifier qu'une publication est complète et inchangée;
- monter plusieurs publications sous des routes stables;
- afficher les publications disponibles dans la barre de navigation générée;
- lister et supprimer les artefacts de publication locaux.

Les versions publiées restent des fichiers statiques. Les lecteurs n'ont pas besoin d'un compte M-Press
compte, de service hébergé ou d'application côté serveur.

## Activer les versions

```yaml
versioning:
  enabled: true
  current: next
  artifactsDir: .mpress/versions
```

## Capturer une publication

```sh
mpress build --strict
mpress check
mpress versions capture v1.0
mpress versions verify v1.0
```

Capture copie la sortie actuelle et écrit `mpress-version.json` contenant un
contrôle SHA-256 pour chaque fichier. `verify` échoue si un fichier capturé est manquant ou
a changé.

Exécutez `mpress build` de nouveau après la capture. Les versions activées sont montées sous
`/versions/<label>/`, `versions.json` est généré, et le site actuel affiche
un sélecteur de version.

```text
site/
├── index.html
└── versions/
    ├── versions.json
    └── v1.0/
        └── index.html
```

Utilisez `mpress versions list` et `mpress versions remove <label>` pour gérer les
artefacts. `capture --force <label>` remplace intentionnellement une étiquette existante.

@note{type="warning" title="Commit or store the artifacts"}
Si les versions doivent survivre à un checkout propre, conservez `.mpress/versions` dans un stockage durable
ou commitez-le selon votre politique de publication.
@end
