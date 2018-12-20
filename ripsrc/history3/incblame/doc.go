// Package incblame generates blame data incrementally based on history of patches. To use first parse the file diff using Parse and then apply resulting Diff to Blame data. You start by calling blame := Apply(nil, diff, "..") and then using new blame data as parents for futher commits. See tests for examples.
package incblame
