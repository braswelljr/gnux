// Vercel's Ignored Build Step (vercel.json): exit 0 skips the build, 1 runs it.
// A Dependabot branch, commit or merge does not deploy, so its updates reach
// the site with the next ordinary commit. Anything unrecognized builds.
const {
  VERCEL_GIT_COMMIT_REF: ref = '',
  VERCEL_GIT_COMMIT_AUTHOR_LOGIN: author = '',
  VERCEL_GIT_COMMIT_MESSAGE: message = ''
} = process.env;

const dependabot =
  ref.startsWith('dependabot/') ||
  author === 'dependabot[bot]' ||
  /^Merge pull request #\d+ from [^/\s]+\/dependabot\//.test(message);

console.log(dependabot ? 'vercel-ignore: Dependabot update, skipping' : 'vercel-ignore: building');
process.exit(dependabot ? 0 : 1);
