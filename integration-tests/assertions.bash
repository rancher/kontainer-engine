# assertions for tests

# output_contains()
#
# assert that the $output variable contains the supplied text
# if the assertion is true, then the $CONTAINS will equal true
# otherwise it will be false
#
output_contains () {
    if [[ "$output" == *"$1"* ]]; then
        contains=true
    fi
}