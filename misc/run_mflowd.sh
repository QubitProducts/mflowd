[ -z "$MFLOWD_VERBOSE" ] || VERBOSE=-v
export GOOGLE_APPLICATION_CREDENTIALS=/etc/mflowd/key.json
./mflowd $VERBOSE -p 6666 -s pubsub $MFLOWD_SUB
