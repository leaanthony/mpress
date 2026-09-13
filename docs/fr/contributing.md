---
title: Contribuer à M-Press
description: Proposez une pull request avec un test qui démontre une lacune de M-Press. Un correctif est également bienvenu.
order: 35
---

Toute contribution à M-Press commence par une pull request et un test qui
démontre une lacune. Nous n'acceptons pas les issues. Si une fonctionnalité est
incorrecte ou manquante, montrez le comportement attendu dans un test reproductible.

Si vous pouvez aussi corriger le problème, tant mieux. Une PR contenant seulement
un test en échec est également bienvenue : elle nous donne un exemple concret.

## Démontrer la lacune

1. Créez un fork de [M-Press](https://github.com/leaanthony/mpress), puis une branche à partir de `main`.
2. Ajoutez le plus petit test qui démontre le comportement incorrect ou manquant.
   Placez-le près du code concerné dans un fichier `*_test.go`, en utilisant les
   fonctions de test et les jeux de données existants du paquet.
3. Exécutez le test sur l'implémentation actuelle. Vérifiez qu'il échoue à cause
   de la lacune, et non d'un problème de configuration.
4. Si vous le pouvez, corrigez l'implémentation et vérifiez que le même test réussit.

Utilisez la version de Go indiquée dans `go.mod` ou une version plus récente.
Les tests publics s'exécutent entièrement depuis ce dépôt. Aucun accès aux suites
de tests privées n'est nécessaire.

## Ouvrir la pull request

Indiquez :

- Le comportement attendu et le comportement observé.
- Le test qui démontre la lacune et la commande exacte pour l'exécuter.
- L'échec observé avant correction et, si vous proposez un correctif, le résultat après correction.

Limitez chaque PR à une seule lacune. Si vous proposez seulement le test en échec,
ouvrez une **PR en brouillon** et précisez qu'un correctif reste nécessaire.
L'échec du test reproduit le problème. La PR pourra être fusionnée après correction
et réussite des vérifications.

Pour un correctif, exécutez les tests des paquets concernés, puis les vérifications publiques :

```sh
go test -race ./...
go vet ./...
```

Si la modification touche la documentation ou le contenu généré, exécutez aussi :

```sh
go run ./cmd/mpress build --strict
go run ./cmd/mpress check
```

## Signaler une vulnérabilité

Signalez toute vulnérabilité présumée en privé selon la
[politique de sécurité](https://github.com/leaanthony/mpress/blob/main/SECURITY.md).
Ne publiez pas d'identifiants ni de détails d'exploitation dans une PR publique.

## Contribuer à un site de documentation

Pour permettre aux lecteurs de contribuer à un site créé avec M-Press, consultez
[Activer les contributions au site](/how-to/enable-site-contributions/).
