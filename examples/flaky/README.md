# Flaky Test Example

This code in here is an example for a flaky test. It can be run to generate some output suitable for testing flaky behavior and the resulting json output.

At the time of writing, Go does not have built-in support for re-running flaky tests, (though there is an [accepted proposal](https://github.com/golang/go/issues/62244) to address this). However, [gotestsum](https://github.com/gotestyourself/gotestsum) does include support to automatically re-run flaky tests, with the `--rerun-fails=n` and `--rerun-fails-max-failures=n` flags.

The output for the unit test can be generated with the following command from the repo root:

```shell
gotestsum --jsonfile examples/flaky/test.json --rerun-fails --packages ./examples/flaky -- -count 1 -tags examples
```
