#!/bin/bash -eu
#
##################################################
# This script is used to generate config settings for ipfs and set the bootstrap list.
# Then run docker-compose command to start the ipfs.
##################################################

CURRENT_PATH=`pwd`

DATA_PATH=$CURRENT_PATH/data
EXPORT_PATH=$CURRENT_PATH/export

CFG_FILE=$DATA_PATH/config


DOCKER_COM_FILE="docker-compose.yml"
function cleanUp()
{
    docker-compose -f $DOCKER_COM_FILE down

}

function generateCFG()
{
    docker run -v $EXPORT_PATH:/export -v $DATA_PATH:/data/ipfs ipfs/go-ipfs:latest --init true >/dev/null 2>&1
}

function replaceBootstrapList()
{

    # TARGET_BOOTSTRAP_PATTERN_1="\"\/ip4\/58.83.177.151\/tcp\/8001\/ipfs\/QmQ4rk6y7AWKCb6VtXkTeeqJMGRgyvw7MJnthkQUA6D3Mg\""
    # TARGET_BOOTSTRAP_PATTERN_2="\"\/ip4\/58.83.177.151\/tcp\/8002\/ipfs\/QmRMrFTrgiQrPmVeGbNWZfjiYUN2SDEa2eyPncTStX2g1H\""
    TARGET_BOOTSTRAP_PATTERN_1="\"\/ip4\/120.92.76.105\/tcp\/8001\/ipfs\/QmQDGhmWuFSBAkB1aVo1t9auMpsL3zNp7y8zTJtJr7tv6i\""
    TARGET_BOOTSTRAP_PATTERN_2="\"\/ip4\/120.92.86.242\/tcp\/8002\/ipfs\/QmUScFSvv5ujngvWyWfzT3xSkSTgzm3KmW5aQnZFUEfAQv\""
    ORIGINAL_BOOTSTRAP_PATTERN_1="\"\/ip4\/.*\/tcp\/4001\/ipfs\/.*\""
    ORIGINAL_BOOTSTRAP_PATTERN_2="\"\/ip6\/.*\/tcp\/4001\/ipfs\/.*\""

    sed -i "s/$ORIGINAL_BOOTSTRAP_PATTERN_1/$TARGET_BOOTSTRAP_PATTERN_1/g" $CFG_FILE
    sed -i "s/$ORIGINAL_BOOTSTRAP_PATTERN_2/$TARGET_BOOTSTRAP_PATTERN_2/g" $CFG_FILE

    if [ $? -ne 0 ]
    then 
        echo "Failed to replace $CFG_FILE."
        exit
    else
        echo "Success to generate $CFG_FILE."
    fi

}

function start()
{
    docker-compose -f $DOCKER_COM_FILE up -d 2>&1
    if [ $? -ne 0 ]
    then 
        echo "Failed to run: docker-compose -f $DOCKER_COM_FILE up -d 2>&1."
        exit
    else
        echo "Successed! IPFS is ready now."
    fi
}


if [ -f "$CFG_FILE" ]
then
    cleanUp
else
    generateCFG
fi

#replaceBootstrapList
start
