#@/bin/sh

# show config
./df --showconfig

# set some values
export DF_MINSIZE=1024
export DF_DBFILE=dupefiles.db
export DF_BINARY_COMPARE_SIZE=0
export DF_DEBUG=true
export DF_DRYRUN=true

# show config
./df --showconfig