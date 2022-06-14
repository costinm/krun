# Ko

Original version of krun and cloud-run-mesh used ko for build. This is no longer the case - ko is great, but too inflexible
and using 'crane' directly is more powerful and less opinionated.

For reference, this is how it worked:


## Github action:

```yaml
jobs:
  build-ko:
    name: Build with ko
    runs-on: ubuntu-latest
    steps:
      - uses: actions/setup-go@v2
        with:
          go-version: 1.16
      - name: What
        run:
          echo "BRANCH=${GITHUB_REF##*/}" >> $GITHUB_ENV
      - uses: actions/checkout@v2
      - uses: imjasonh/setup-ko@v0.4
      - run: ko publish -t ${{ env.BRANCH }} -B ./

```
