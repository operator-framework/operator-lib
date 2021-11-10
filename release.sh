#!/usr/bin/env bash

###############################################################################################
### RELEASING OPERATOR-LIB
# Every operator-lib release should have a corresponding git semantic version tag
# begining with `v`, ex.n `v1.2.3`.
#
# STEP 1: Create a release branch with the name vX.Y.x. Example: git checkout -b v1.2.x
#
# STEP 2: Run the release script by providing the operator-lib release version as an argument
# in the above mentioned format. Example: ./release vX.Y.Z
#
# STEP 3: This script will create a release tag locally. Push the release branch and tag:
# git push upstream <release-branch>
# git push upstream <tag-name>, wherein <tag-name> is the release version.
#
# STEP 4: Update the release notes in github with the changes included in corresponding
# operator-lib version.
#################################################################################################

set -eu

if [[ $# != 1 ]]; then
	echo "usage: $0 vX.Y.Z"
	exit 1
fi

VER=$1
NUMRE="0|[1-9][0-9]*"
PRERE="\-(alpha|beta|rc)\.[1-9][0-9]*"

if ! [[ "$VER" =~ ^v($NUMRE)\.($NUMRE)\.($NUMRE)($PRERE)?$ ]]; then
	echo "malformed version: \"$VER\""
	exit 1
fi

if ! git diff-index --quiet HEAD --; then
	echo "directory has uncommitted files"
	exit 1
fi

# Run tests
echo "Running tests"
make check

# Tag the release commit and verify its tag
echo "Creating a new tag for Operator-lib version $VER"
git tag --sign --message "operator-lib $VER" "$VER"
git verify-tag --verbose $VER

# Add reminder on next stpes
echo ""
echo "Done forget to:"
echo ""
echo "git push upstream <release-branch>"
echo "git push upstream $VER"
echo ""
echo "Also update the release notes in github for this tag."
