function parseList(value) {
  return (value || '').trim().split('\n').map((entry) => entry.trim()).filter(Boolean);
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

// Reads config once from workflow env vars and builds the header regex.
function buildRules() {
  const rawTypes = parseList(process.env.COMMIT_TYPES);
  if (rawTypes.length === 0) throw new Error('COMMIT_TYPES must not be empty.');

  const allowedScopes = parseList(process.env.ALLOWED_SCOPES);
  const parsedMaxLength = parseInt(process.env.MAX_SUBJECT_LENGTH, 10);
  const maxLength = Number.isNaN(parsedMaxLength) ? 72 : parsedMaxLength;
  const lowercaseOnly = process.env.SUBJECT_CASE !== 'any';
  const allowBang = process.env.ALLOW_BREAKING_CHANGE_MARKER !== 'false';

  // type(scope)!: description — scope and the breaking-change "!" are both optional.
  const escapedTypes = rawTypes.map(escapeRegExp).join('|');
  const pattern = new RegExp(
    `^(?:${escapedTypes})(?:\\(([^)]+)\\))?${allowBang ? '!?' : ''}: ${lowercaseOnly ? '[a-z]' : '.'}.+$`
  );

  return { rawTypes, allowedScopes, maxLength, lowercaseOnly, pattern };
}

// Checks one header line (a PR title or a commit subject) against the rules.
// Returns a list of problem descriptions; empty means it's valid.
function checkHeader(header, rules) {
  const match = header.match(rules.pattern);
  if (!match) return ['does not match the Conventional Commits format'];

  const problems = [];

  if (header.length > rules.maxLength) {
    problems.push(`is ${header.length} characters, exceeds the ${rules.maxLength} character limit`);
  }

  const scope = match[1];
  if (scope && rules.allowedScopes.length > 0) {
    const givenScopes = scope.split(',').map((s) => s.trim());
    const unknownScopes = givenScopes.filter(
      (s) => !rules.allowedScopes.some((pattern) => new RegExp(`^${pattern}$`).test(s))
    );
    if (unknownScopes.length > 0) {
      const label = unknownScopes.length > 1 ? 'scopes' : 'scope';
      problems.push(`uses ${label} "${unknownScopes.join(', ')}", which ${unknownScopes.length > 1 ? 'are' : 'is'} not allowed`);
    }
  }

  return problems;
}

function describeRules(rules) {
  const lines = [
    'Expected: <type>(<optional scope>): <description>',
    `Types: ${rules.rawTypes.join(', ')}`,
    `Max length: ${rules.maxLength}`,
  ];
  if (rules.lowercaseOnly) lines.push('The description must start with a lowercase letter.');
  if (rules.allowedScopes.length > 0) lines.push(`Allowed scopes: ${rules.allowedScopes.join(', ')}`);
  return lines.join('\n');
}

async function validatePrTitle({ github, context, core }) {
  const rules = buildRules();

  // Re-fetch the PR so this checks the current title, not a possibly-stale
  // value from the event payload (e.g. on a manual re-run after a title edit).
  const { data: pr } = await github.rest.pulls.get({
    owner: context.repo.owner,
    repo: context.repo.repo,
    pull_number: context.payload.pull_request.number,
  });

  const problems = checkHeader(pr.title, rules);
  if (problems.length > 0) {
    core.setFailed(`PR title "${pr.title}" ${problems.join(' and ')}.\n\n${describeRules(rules)}`);
  }
}

async function validateCommitMessages({ github, context, core }) {
  const rules = buildRules();

  const commits = await github.paginate(github.rest.pulls.listCommits, {
    owner: context.repo.owner,
    repo: context.repo.repo,
    pull_number: context.payload.pull_request.number,
  });

  const errors = [];
  for (const item of commits) {
    if (item.parents && item.parents.length > 1) continue; // skip merge commits

    const subject = item.commit.message.split('\n')[0].replace(/\r$/, ''); // first line, strip CR
    const problems = checkHeader(subject, rules);
    if (problems.length > 0) {
      errors.push(`${item.sha.substring(0, 7)}: "${subject}" ${problems.join(' and ')}.`);
    }
  }

  if (errors.length > 0) {
    core.setFailed(
      'The following commits do not follow Conventional Commits format:\n' +
      errors.join('\n') + '\n\n' + describeRules(rules)
    );
  }
}

module.exports = { validatePrTitle, validateCommitMessages };
