#!/bin/sh
# Cloud Run Jobs configuration script

jobs=(
    "abu '-abugames'"
    "blueprint '-blueprint'"
    "cardsphere '-cardsphere'"
    "cardtrader '-cardtrader'"
    "ck '-cardkingdom'"
    "cksealed '-cardkingdom_sealed'"
    "csi '-coolstuffinc'"
    "csiofficial '-coolstuffinc_official'"
    "hareruyabuylist '-vendors hareruya'"
    "hareruyainv '-sellers hareruya'"
    "jupiter '-jupitergames'"
    "magiccorner '-magiccorner'"
    "mkm '-mkm_index'"
    "mkmsealed '-mkm_sealed'"
    "mtgseattle '-mtgseattle'"
    "mtgstocks '-mtgstocks'"
    "mythic '-mythicmtg'"
    "ninetyfive '-ninetyfive'"
    "scg '-starcitygames'"
    "strikezone '-strikezone'"
    "tcgbuylist '-tcg_market'"
    "tcgindex '-tcg_index'"
    "tcgsealed '-tcg_sealed'"
    "tcgsyp '-tcgplayer_syp'"
    "tnt '-trollandtoad'"
    "toa '-talesofadventure'"
    "wizcupboard '-wizardscupboard'"
)

# Execute bantool binary with generated command line flags
for job in "${jobs[@]}"; do
    # split variables for concurrency
    job_name="${job%% *}"
    flags="${job#* }"

    export OUTPUT_PATH="gs://mtgbanzai/$job_name"
    eval "./bantool -svc-acc '/tmp/cloudrunner' -format 'ndjson' -output-path '$OUTPUT_PATH' $flags"
done