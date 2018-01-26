# setup and teardown for tests
setup_environment() {
    rm -r .home/
    mkdir .home/

    # load mock azure service definition
    hoverctl start webserver > /dev/null 2>&1
    hoverctl import ./integration-tests/aks/azure-api.json > /dev/null 2>&1
}

teardown_environment() {
    hoverctl stop > /dev/null 2>&1
}
