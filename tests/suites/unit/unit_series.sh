run_unit_set_series() {
	# Echo out to ensure nice output to the test suite.
	echo

	# The following ensures that a bootstrap juju exists.
	file="${TEST_DIR}/test-unit-series.log"
	ensure "unit-series" "${file}"

	echo "Deploy ubuntu focal"
	juju deploy ubuntu --series=focal

	wait_for "ubuntu" "$(idle_condition "ubuntu")"

	echo "Change application base to jammy and add-unit"
	juju set-application-base ubuntu jammy
	juju add-unit ubuntu

	wait_for "ubuntu" "$(idle_condition "ubuntu" 0 1)"

	echo "Check the base for machine of added unit"
	juju status --format=json | jq -r '.machines | .["1"] | .base | .name' | grep "ubuntu"
	juju status --format=json | jq -r '.machines | .["1"] | .base | .channel' | grep "22.04"

	destroy_model "unit-series"
}

test_unit_series() {
	if [ -n "$(skip 'test_unit_series')" ]; then
		echo "==> SKIP: Asked to skip unit series tests"
		return
	fi

	(
		set_verbosity

		cd .. || exit

		run "run_unit_set_series"
	)
}
