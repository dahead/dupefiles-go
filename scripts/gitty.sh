#!bin/bash
# cd /pfad/zum/lokalen/projekt
git init
git add .
git commit -m "Initial commit mit aktuellem lokalen Stand"
git remote add origin https://github.com/dahead/dupefiles-go
git branch -M main
git push -u origin main --force

