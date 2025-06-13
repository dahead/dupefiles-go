#@/bin/sh

# show config
./df --showconfig

# set some values
export DF_MINSIZE=1024
export DF_DBFILE=dupefiles.db

# start scan
./df --quickscan ~/Photos