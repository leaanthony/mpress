---
title: Composants de données et de produit
description: Présentez des comparaisons, des versions, l'état, les dates, les plans et des preuves.
order: 8
---

## Matrice de comparaison

@matrix{highlight="M-Press"}
| Capacité | M-Press | SSG Node typique |
| --- | --- | --- |
| Binaire unique | ✓ | ✗ |
| Markdown et HTML | ✓ | ✓ |
| Environnement d'exécution requis | ✗ | ✓ |
| Versions intégrées | ✓ | ~ |
@end

@details{title="Source"}
```md
@matrix{highlight="M-Press"}
| Capacité | M-Press | SSG Node typique |
| --- | --- | --- |
| Binaire unique | ✓ | ✗ |
| Markdown et HTML | ✓ | ✓ |
| Environnement d'exécution requis | ✗ | ✓ |
| Versions intégrées | ✓ | ~ |
@end
```
@end

## État du service

@status{service="Build service" state="operational"}
Tous les systèmes sont opérationnels
@end

@status{service="Translation queue" state="degraded"}
Les tâches peuvent prendre plus de temps que d'habitude
@end

@details{title="Source"}
```md
@status{service="Build service" state="operational"}
Tous les systèmes sont opérationnels
@end

@status{service="Translation queue" state="degraded"}
Les tâches peuvent prendre plus de temps que d'habitude
@end
```
@end

## Calendrier

@calendar{month="2026-08" style="compact"}
Utilisez les flèches pour parcourir les mois.
@end

@details{title="Source"}
```md
@calendar{month="2026-08" style="compact"}
Utilisez les flèches pour parcourir les mois.
@end
```
@end

## Journal des modifications

@changelog
### v0.2.0 (2026-08-01)
#### Ajouts
- Bibliothèque de composants complète
- Copie interactive du terminal
#### Corrections
- Alignement des icônes du thème
- Rythme de la table des matières

### v0.1.0 (2026-07-30)
#### Ajouts
- Compilateur de site statique initial
@end

@details{title="Source"}
```md
@changelog
### v0.2.0 (2026-08-01)
#### Ajouts
- Bibliothèque de composants complète
- Copie interactive du terminal
#### Corrections
- Alignement des icônes du thème
- Rythme de la table des matières

### v0.1.0 (2026-07-30)
#### Ajouts
- Compilateur de site statique initial
@end
```
@end

## Notes de version

@release{version="0.2.0" date="2026-08-01" type="minor"}
### Points forts
- M-Press documente maintenant chaque composant pris en charge.
### Nouvelles fonctionnalités
- Rendus pour le terminal, le diff, l'API, le calendrier, la tarification et les tutoriels.
### Corrections de bogues
- Espacement cohérent des composants et couleurs du mode sombre.
@end

@details{title="Source"}
```md
@release{version="0.2.0" date="2026-08-01" type="minor"}
### Points forts
- M-Press documente maintenant chaque composant pris en charge.
### Nouvelles fonctionnalités
- Rendus pour le terminal, le diff, l'API, le calendrier, la tarification et les tutoriels.
### Corrections de bogues
- Espacement cohérent des composants et couleurs du mode sombre.
@end
```
@end

## Tarification

Le composant de tarification affiche des offres fictives. Le paiement et la
gestion des abonnements nécessitent un service externe.

@pricing{cols="2"}
### Personnel
$0
[Commencer](/getting-started/)
- ✓ Un projet
- ✓ Assistance communautaire
---
### Équipe
$15/mois
[En savoir plus](/data-components/)
recommended
- ✓ Projets partagés
- ✓ Assistance prioritaire
@end

@details{title="Source"}
```md
@pricing{cols="2"}
### Personnel
$0
[Commencer](/getting-started/)
- ✓ Un projet
- ✓ Assistance communautaire
---
### Équipe
$15/mois
[En savoir plus](/data-components/)
recommended
- ✓ Projets partagés
- ✓ Assistance prioritaire
@end
```
@end

## Témoignages

@testimonials{autoplay="0"}
“Le contenu reste du Markdown ordinaire.”
Auteur de la documentation
---
“La sortie fonctionne sans framework client.”
Ingénieur plateforme
@end

@details{title="Source"}
```md
@testimonials{autoplay="0"}
“Le contenu reste du Markdown ordinaire.”
Auteur de la documentation
---
“La sortie fonctionne sans framework client.”
Ingénieur plateforme
@end
```
@end

## Codes QR

Les codes QR sont générés dans la page au moment de la génération. Il n'y a pas de
service d'image externe.

@qr{url="https://github.com/leaanthony/mpress" size="120" label="M-Press repository"}

@details{title="Source"}
```md
@qr{url="https://github.com/leaanthony/mpress" size="120" label="M-Press repository"}
```
@end
