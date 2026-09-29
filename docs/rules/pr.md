## PR instructions

After completing work on a feature or fix branch, create a pull request into
`main`. Before creating it:

1. Verify the working tree is clean and review the branch diff against `main`.
2. Run the relevant validation commands for every affected application and
   record their results.
3. Push the current branch and create a PR with a Conventional Commits title
   matching the primary change (for example, `feat: add monitor creation`).
4. Use a detailed PR body containing these sections: `Summary`, `Changes`,
   `Validation`, `Security and architecture`, `Affected areas`, and
   `Screenshots` (or state that no visual evidence is applicable).
5. Include links to related issues when they exist. Do not create, merge, or
   close a PR without the user's explicit request.

For visible UI changes, attach screenshots or a recording to the PR. If the
repository has no configured GitHub remote or the GitHub account is not
connected, report that blocker and preserve the prepared title and body for
publication after access is available.
