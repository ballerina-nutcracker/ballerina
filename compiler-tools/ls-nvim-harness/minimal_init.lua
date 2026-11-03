-- minimal_init.lua
--
-- Headless-Neovim harness for the Ballerina Go language server. Attaches
-- Neovim's real built-in vim.lsp client to `bal start-language-server`, so an
-- agent can drive genuine editor traffic (real capability negotiation, real
-- document sync from real buffer edits) one command at a time via
-- `nvim --server <sock> --remote-expr 'v:lua.Harness.<fn>(...)'`.
--
-- Not a test suite: there is no assertion framework here (no busted/plenary).
-- Every Harness.* function returns a JSON string (via vim.json.encode) so the
-- caller on the other side of --remote-expr gets plain JSON to read, not a
-- Lua table to parse.

vim.lsp.log.set_level(vim.lsp.log.levels.OFF)

-- Minimal base64 decoder (standard algorithm) so free-text payloads (edit
-- text, raw request params) can cross the `--remote-expr` shell boundary
-- without quoting hazards. probe.sh base64-encodes; this reverses it.
local B64 = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/'
local function base64_decode(data)
  data = string.gsub(data, '[^' .. B64 .. '=]', '')
  return (data:gsub('.', function(x)
    if x == '=' then return '' end
    local r, f = '', (B64:find(x) - 1)
    for i = 6, 1, -1 do r = r .. (f % 2 ^ i - f % 2 ^ (i - 1) > 0 and '1' or '0') end
    return r
  end):gsub('%d%d%d?%d?%d?%d?%d?%d?', function(x)
    if #x ~= 8 then return '' end
    local c = 0
    for i = 1, 8 do c = c + (x:sub(i, i) == '1' and 2 ^ (8 - i) or 0) end
    return string.char(c)
  end))
end

local bal_cmd = os.getenv('LS_HARNESS_BAL') or 'bal'
local root_dir = os.getenv('LS_HARNESS_ROOT') or vim.fn.getcwd()

local client_id = nil

-- uri -> { diagnostics = {...}, version = N }, bumped on every
-- textDocument/publishDiagnostics push so Harness.diagnostics can poll for a
-- fresh result instead of returning a stale cache immediately after an edit.
local diagnostics_cache = {}
local diagnostics_seq = 0

local default_publish_diagnostics = vim.lsp.handlers['textDocument/publishDiagnostics']
vim.lsp.handlers['textDocument/publishDiagnostics'] = function(err, result, ctx, config)
  if result and result.uri then
    diagnostics_seq = diagnostics_seq + 1
    diagnostics_cache[result.uri] = { diagnostics = result.diagnostics, seq = diagnostics_seq }
  end
  return default_publish_diagnostics(err, result, ctx, config)
end

local function ensure_client()
  if client_id then
    return client_id
  end
  client_id = vim.lsp.start({
    name = 'ballerina-go-ls',
    cmd = { bal_cmd, 'start-language-server' },
    root_dir = root_dir,
  }, { attach = false })
  if not client_id then
    error('failed to start ' .. bal_cmd .. ' start-language-server')
  end
  return client_id
end

-- Resolves a file path to a loaded, LS-attached buffer. Idempotent.
local function open_buf(path)
  local abspath = vim.fn.fnamemodify(path, ':p')
  local bufnr = vim.fn.bufadd(abspath)
  vim.fn.bufload(bufnr)
  local id = ensure_client()
  if not vim.lsp.buf_is_attached(bufnr, id) then
    vim.lsp.buf_attach_client(bufnr, id)
  end
  return bufnr
end

local function position_params(bufnr, line, col)
  return {
    textDocument = { uri = vim.uri_from_bufnr(bufnr) },
    position = { line = line, character = col },
  }
end

local function request_sync(bufnr, method, params, timeout_ms)
  local results = vim.lsp.buf_request_sync(bufnr, method, params, timeout_ms or 2000)
  if not results then
    return { error = 'timeout' }
  end
  -- buf_request_sync keys by client_id; there is only ever one client here.
  for _, res in pairs(results) do
    return res.result ~= nil and res.result or res.err
  end
  return vim.NIL
end

Harness = {}

function Harness.open(path)
  local bufnr = open_buf(path)
  return vim.json.encode({ bufnr = bufnr, uri = vim.uri_from_bufnr(bufnr) })
end

function Harness.hover(path, line, col)
  local bufnr = open_buf(path)
  local result = request_sync(bufnr, 'textDocument/hover', position_params(bufnr, line, col))
  return vim.json.encode(result)
end

function Harness.definition(path, line, col)
  local bufnr = open_buf(path)
  local result = request_sync(bufnr, 'textDocument/definition', position_params(bufnr, line, col))
  return vim.json.encode(result)
end

function Harness.completion(path, line, col)
  local bufnr = open_buf(path)
  local result = request_sync(bufnr, 'textDocument/completion', position_params(bufnr, line, col))
  return vim.json.encode(result)
end

-- Diagnostics arrive as an unsolicited push, not a request/response, so this
-- polls the cache for up to timeout_ms for an update newer than the call.
function Harness.diagnostics(path, timeout_ms)
  local bufnr = open_buf(path)
  local uri = vim.uri_from_bufnr(bufnr)
  local seq_before = (diagnostics_cache[uri] or {}).seq or 0
  vim.wait(timeout_ms or 1000, function()
    return ((diagnostics_cache[uri] or {}).seq or 0) > seq_before
  end, 50)
  local entry = diagnostics_cache[uri]
  return vim.json.encode(entry and entry.diagnostics or {})
end

-- Reads Neovim's own vim.diagnostic store for the buffer -- confirms
-- publishDiagnostics reached the editor's native diagnostic system, not
-- just the JSON cache above.
function Harness.native_diagnostics(path)
  local bufnr = open_buf(path)
  local diags = vim.diagnostic.get(bufnr)
  local out = {}
  for _, d in ipairs(diags) do
    table.insert(out, {
      lnum = d.lnum, col = d.col, end_lnum = d.end_lnum, end_col = d.end_col,
      severity = d.severity, message = d.message, source = d.source, code = d.code,
    })
  end
  return vim.json.encode(out)
end

-- Real buffer edit, not a hand-crafted didChange -- Neovim's attached client
-- turns this into genuine document-sync traffic.
function Harness.edit(path, start_line, start_col, end_line, end_col, text_b64)
  local bufnr = open_buf(path)
  local text = base64_decode(text_b64)
  local lines = vim.split(text, '\n', { plain = true })
  vim.api.nvim_buf_set_text(bufnr, start_line, start_col, end_line, end_col, lines)
  return vim.json.encode({ ok = true })
end

-- Whole-buffer replace, for resetting content without tracking exact
-- end-line/end-col (nvim_buf_set_lines(-1) means "to the last line").
function Harness.set_text(path, text_b64)
  local bufnr = open_buf(path)
  local text = base64_decode(text_b64)
  local lines = vim.split(text, '\n', { plain = true })
  vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, lines)
  return vim.json.encode({ ok = true })
end

-- Escape hatch: arbitrary method/params for anything not wrapped above yet.
function Harness.raw(path, method, params_b64)
  local bufnr = open_buf(path)
  local params = vim.json.decode(base64_decode(params_b64))
  local result = request_sync(bufnr, method, params)
  return vim.json.encode(result)
end

function Harness.stop()
  if client_id then
    local client = vim.lsp.get_client_by_id(client_id)
    if client then
      client:stop(true)
    end
  end
  vim.schedule(function()
    vim.cmd('qa!')
  end)
  return vim.json.encode({ ok = true })
end
