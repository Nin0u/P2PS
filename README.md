# ProtInt
# Projet Protocole Internet


## Compilation

Dans le dossier main
```
go build
```

## Lancement

Dans le dossier main
```
./main --username="votre nom"
```

D'autre options sont possible :
- ``--gui`` pour lancer l'interface graphique
- ``--export=path/to/folder`` pour exporter l'arborescence contenu le dossier 'path/to/folder' 
- ``--debug`` pour avoir les logs

## Information Complémentaire
Le projet dispose de 2 interfaces : un CLI et un GUI.
Les fichiers téléchargés avec le CLI sont stockés dans le dossier ``DOWNLOAD``.
Quelques touches importantes sur le CLI : 
- ``ESC`` pour quitter le porgramme
- ``ARROW_UP/ARROW_DOWN`` pour naviguer dans l'historique des commandes executées 
Attention, le CLI étant fait à la va vite, il peut y avoir des soucis si les arguments des commandes contiennent des espaces. Nous recommandons plutôt d'utiliser l'interface graphique. De plus la bibliothèque qu'on utilise pour récupérer sur quelle touche appuie l'utilisateur a quelques bugs, et il se peut que les touches ne soient plus reconnues, donc relancer votre terminal à la dur pour arrêter le bug.
