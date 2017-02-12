# Major release
Major version releases come from the feature-freeze branch, and are merged into master, **only** when API compatibility
changes are made. Release candidates are drafted until it is believe the changes are stable, and can then be release and merged
back into the master branch. The number of release candidates is unlimited (i.e until it's stable), but should probably be low.
1. Increment major version number in VERSION

2. Write release notes in CHANGELOG.md

- (**be sure to make backwards incompatible changes very clear**)

3. Merge from Feature-freeze to master

4. Make git tag

5. Make github release

6. Merge from master back to dev

# Minor release
Minor version releases come from the dev branch, and are merged into feature-freeze 
when backwards compatible API changes are made, features or bugfixes are added.

1. Increment minor version number in VERSION

2. Write release notes in CHANGELOG.md

3. Merge from dev to feature-freeze

4. Make git tag

5. Make github release

# Patch release
Minor patch. Should be per change, regardless of severity, size or type.
1. Increment patch release number in VERSION

2. Write release notes in CHANGELOG.md

4. Make git tag
