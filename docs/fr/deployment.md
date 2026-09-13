---
title: Déploiement
description: Publier la sortie de M-Press sur n'importe quel hôte statique.
order: 10
---

Un build M-Press est un répertoire de fichiers statiques. Aucun processus M-Press, runtime Go,
base de données ni serveur Node.js n'est requis après le build.

## Checklist de publication

```sh
mpress build --strict
mpress check
```

Publiez le répertoire de sortie configuré, `site/` par défaut, comme racine web.

## Déployer depuis M-Press

M-Press peut créer un projet Cloudflare Pages et publier la build statique complète
build. Configurez la cible une seule fois :

```sh
mpress deploy configure cloudflare \
  --account YOUR_ACCOUNT_ID \
  --project your-documentation
```

Conservez les identifiants en dehors `mpress.yaml`:

```sh
export CLOUDFLARE_API_TOKEN=your-token
```

Le jeton doit avoir l'autorisation d'éditer les projets Cloudflare Pages. Créez une prévisualisation
d'abord, puis publiez la même sortie vérifiée lorsqu'elle est prête :

```sh
mpress deploy
mpress deploy --production
```

Netlify utilise un ID de site, un domaine ou un nom de site existant. Indiquez un team slug uniquement
lorsque le site appartient à une équipe :

```sh
mpress deploy configure netlify \
  --site docs.example.com \
  --account your-team
export NETLIFY_AUTH_TOKEN=your-token
mpress deploy
mpress deploy --production
```

Les previews Netlify sont des déploiements atomiques en brouillon. Ils ne remplacent pas le site publié
site. Les tokens Netlify restent dans l'environnement et ne sont jamais écrits dans
`mpress.yaml`.

En développement, sélectionnez **Déployer** depuis le menu M-Press en bas du
site généré. Choisissez Cloudflare Pages ou Netlify. Le formulaire modifie la même
configuration et exécute les mêmes vérifications de publication que la ligne de commande. Il n'
écrit un token d'API dans le projet.

## URL de base et sitemap

Définissez l'origine publique avant un build de production :

```yaml
site:
  baseURL: https://docs.example.com
```

M-Press génère ensuite `sitemap.xml` en utilisant cette origine. Les routes restent des URLs de répertoire
se terminant par `/`, donc les hôtes doivent servir chaque répertoire `index.html`.

## Choisir un hôte

Utilisez n'importe quel hôte qui sert des fichiers statiques :

- Pour GitHub Pages, téléversez `site/` avec un workflow Pages.
- Pour Cloudflare Pages, publiez `site/` ou téléversez une build terminée.
- Pour le stockage d'objets, copiez `site/` dans le bucket et configurez le CDN.
- Pour votre propre serveur, copiez `site/` dans la racine web pour Apache, Caddy ou Nginx.

Configurez l'hôte pour servir `index.html` pour les routes de répertoire. Si vous publiez le
site sous un préfixe de chemin, assurez-vous que les liens relatifs à la racine utilisent l'origine correcte.

Parce que la recherche est un index JSON statique, elle fonctionne sur chacun de ces hôtes
sans service de recherche côté serveur.
