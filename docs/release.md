# Release Process

Github Actions is the primary release tool, most of the heavy lifting is done in the following workflow: `.github/workflows/release.yaml`

## Starting the process

To start a new release, we first need to create a branch using the following convention: `release-vX.Y.Z`

Once this branch has been created, GitHub actions will update all relevant version numbers to the appropriate branch version.

This will then raise a PR for review.

Once reviewed and approved, a contributor must checkout `main` and tag the `main` branch with the relevant tag from the branch/PR.

This will trigger the full release workflow and push container images and create the GitHub release too.
