const fs = require("fs");

const MARKER = "<!-- benchmark-http-pr-comment -->";

module.exports = async ({ github, context, core }) => {
  // PR identity must come from the trusted workflow_run event context,
  // never from benchmark-meta.json — that file is written by the prepare
  // job while running PR-head code, so a malicious PR could forge it to
  // make this privileged (pull-requests: write) job comment on an
  // unrelated PR. pull_requests is only populated for same-repo PRs and
  // can be ambiguous in theory, so require exactly one association.
  const pulls = context.payload.workflow_run.pull_requests || [];
  if (pulls.length !== 1) {
    core.info(`Skipping: workflow_run reports ${pulls.length} associated pull request(s); expected exactly 1.`);
    return;
  }
  const issueNumber = pulls[0].number;
  const workflowHeadSha = context.payload.workflow_run.head_sha;

  // Bail out if a newer push has already superseded this run.
  const { data: pr } = await github.rest.pulls.get({
    ...context.repo,
    pull_number: issueNumber,
  });
  if (pr.head.sha !== workflowHeadSha) {
    core.info(`Skipping: workflow_run head ${workflowHeadSha} is not current PR head ${pr.head.sha}`);
    return;
  }

  // The artifact itself (report/log) is still untrusted display content
  // from PR-head code — escaped below — but no longer drives *who* we
  // comment on. It may be absent if the prepare run produced none.
  if (!fs.existsSync("benchmark-artifact")) {
    core.info("No benchmark artifact found; nothing to comment.");
    return;
  }

  function readIfExists(path) {
    return fs.existsSync(path) ? fs.readFileSync(path, "utf8") : "";
  }

  // report.md/run.log come from a tool built on PR-head code, so treat
  // them as untrusted: neutralise raw HTML (angle brackets, ampersand)
  // before posting, while leaving the Markdown table readable.
  function escapeHtml(s) {
    return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
  }

  let report = readIfExists("benchmark-artifact/report.md").trim();
  if (!report) {
    const log = readIfExists("benchmark-artifact/run.log").trim();
    report = "```text\n" + (log || "No benchmark output was produced.") + "\n```";
  }
  report = escapeHtml(report);

  const body = [
    MARKER,
    "## HTTP Benchmark — hello service",
    "",
    report,
    "",
    "_Throughput on GitHub-hosted runners is noisy (~5–15% run-to-run); only large deltas are meaningful. Base and head are measured back-to-back on the same runner (`bal run`), so common-mode noise mostly cancels in the delta. Generated for `origin/<base>` vs `HEAD`._",
  ].join("\n");

  // Upsert: update the existing marker comment or create a new one. Require
  // both the marker AND that we (github-actions[bot], the only identity this
  // job ever posts as) authored it — otherwise anyone who can comment on the
  // PR could plant the marker text first and have us overwrite their comment
  // instead of posting our own.
  const comments = await github.paginate(github.rest.issues.listComments, {
    ...context.repo,
    issue_number: issueNumber,
  });
  const existing = comments.find(c => c.body?.includes(MARKER) && c.user?.login === "github-actions[bot]");

  if (existing) {
    await github.rest.issues.updateComment({
      ...context.repo,
      comment_id: existing.id,
      body,
    });
    core.info(`Updated comment ${existing.id}`);
  } else {
    await github.rest.issues.createComment({
      ...context.repo,
      issue_number: issueNumber,
      body,
    });
    core.info("Created HTTP benchmark comment");
  }
};
