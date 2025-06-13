#!/usr/bin/bash
git init
git add .
git commit -m "Initial commit"
git remote add origin https://github.com/dahead/dupefiles-go
git branch -M main
git push -u origin main --force