[ -z "$MFLOWD_VERBOSE" ] || VERBOSE=-v
./mflowd $VERBOSE -p 6666 -s pubsub $MFLOWD_SUB
