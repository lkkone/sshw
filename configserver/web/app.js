const $ = (selector) => document.querySelector(selector);
const $$ = (selector) => Array.from(document.querySelectorAll(selector));

const translations = {
  "SSHW CONFIG CENTER": "SSHW 配置中心",
  "Your connections,": "所有连接，",
  "in one trusted place.": "尽在一个可信空间。",
  "Edit hosts visually, review the generated YAML, then publish a version every computer can pull safely.": "可视化编辑主机、预览生成的 YAML，再发布供每台电脑安全同步的配置版本。",
  "Username": "用户名",
  "Password": "密码",
  "Sign in": "登录",
  "Credentials stay on your server": "凭据仅保存在你的服务器",
  "Config Center": "配置中心",
  "Profile": "配置集",
  "Saved": "已保存",
  "Save draft": "保存草稿",
  "Publish": "发布",
  "CONNECTIONS": "连接配置",
  "Host library": "主机列表",
  "＋ Host": "＋ 主机",
  "＋ Group": "＋ 分组",
  "No connections yet": "还没有连接",
  "Add your first host or group to begin.": "添加第一个主机或分组即可开始。",
  "Sync devices": "同步设备",
  "Version history": "版本历史",
  "Import configuration": "导入配置",
  "NEW CONNECTION": "新建连接",
  "Add host": "添加主机",
  "Add group": "添加分组",
  "Add nested host": "添加组内主机",
  "Add nested group": "添加子分组",
  "Enter the essentials first. You can add authentication and advanced options afterwards.": "先填写必要信息，创建后可继续补充身份验证和高级选项。",
  "Enter a display name.": "请输入显示名称。",
  "Enter a hostname or IP.": "请输入主机名或 IP。",
  "Enter a valid port.": "请输入 1 到 65535 之间的端口。",
  "IMPORT CONFIGURATION": "导入配置",
  "Import an existing sshw file": "导入已有的 sshw 配置",
  "Select your current": "选择当前的",
  "or another YAML file. It will be validated and opened as an unsaved draft for review.": "或其他 YAML 文件。系统会先校验，然后作为未保存草稿打开供你确认。",
  "Choose a configuration file": "选择配置文件",
  "or drag and drop it here · up to 10 MB": "或将文件拖到这里 · 最大 10 MB",
  "How should it be imported?": "选择导入方式",
  "Replace current draft": "替换当前草稿",
  "Best for moving an existing ~/.sshw into the configuration center.": "适合将现有 ~/.sshw 整体迁移到配置中心。",
  "Append to current list": "追加到当前列表",
  "Keep current entries and add everything from this file.": "保留当前条目，并追加文件中的全部配置。",
  "Validate and import": "校验并导入",
  "Choose a file first.": "请先选择配置文件。",
  "The selected file is larger than 10 MB.": "所选文件超过 10 MB。",
  "Replace the current draft with this imported file?": "确定用导入文件替换当前草稿吗？",
  "Imported {hosts} hosts and {groups} groups. Review the draft, then save it.": "已导入 {hosts} 个主机和 {groups} 个分组。请检查后保存草稿。",
  "Imported entries need attention before they can be saved.": "导入的条目需要处理校验问题后才能保存。",
  "Discard unsaved changes and sign out?": "放弃未保存的更改并退出登录吗？",
  "READY WHEN YOU ARE": "随时可以开始",
  "Select a connection": "选择一个连接",
  "Choose a host from the left, or create one to start building your shared configuration.": "从左侧选择主机，或新建一个连接来维护共享配置。",
  "Create first host": "创建第一个主机",
  "Connections": "连接配置",
  "Host": "主机",
  "New host": "新建主机",
  "Connection details and authentication": "连接信息与身份验证",
  "Identity": "基本信息",
  "How this entry appears in sshw.": "设置这个条目在 sshw 中的显示方式。",
  "Display name": "显示名称",
  "Alias": "别名",
  "Use with": "使用命令",
  "Connection": "连接信息",
  "Remote endpoint used by SSH and SFTP.": "SSH 与 SFTP 使用的远程地址。",
  "Hostname or IP": "主机名或 IP",
  "User": "用户",
  "Port": "端口",
  "Authentication": "身份验证",
  "Leave secrets empty to use an interactive prompt.": "密钥信息留空时，将在连接过程中交互输入。",
  "Private key path": "私钥路径",
  "SSH agent socket": "SSH Agent 套接字",
  "Show": "显示",
  "Key passphrase": "私钥口令",
  "Jump hosts": "跳板机",
  "Optional ordered hops before the destination.": "连接目标主机前按顺序经过的可选跳板机。",
  "＋ Add hop": "＋ 添加跳板机",
  "After login": "登录后执行",
  "Commands sent when an interactive SSH shell opens.": "交互式 SSH Shell 打开后自动发送的命令。",
  "＋ Add command": "＋ 添加命令",
  "Group members": "分组成员",
  "Organize hosts and nested groups together.": "集中组织主机与嵌套分组。",
  "＋ Add host": "＋ 添加主机",
  "＋ Add nested group": "＋ 添加子分组",
  "GENERATED CONFIG": "生成的配置",
  "YAML preview": "YAML 预览",
  "Waiting": "等待输入",
  "Add a host to generate a valid configuration.": "添加主机后即可生成有效配置。",
  "Hosts": "主机",
  "Groups": "分组",
  "Aliases": "别名",
  "PUBLISH CONFIGURATION": "发布配置",
  "Make this version available?": "确认发布这个版本？",
  "Connected computers will receive it the next time they run": "已连接的电脑下次运行",
  ".": "时将收到这个版本。",
  "Release note": "版本说明",
  "Cancel": "取消",
  "Publish version": "发布版本",
  "SYNC ACCESS": "同步授权",
  "Connected devices": "已连接设备",
  "Create a read-only token for each computer. Revoking one device does not affect the others.": "为每台电脑创建独立的只读令牌，吊销其中一台不会影响其他设备。",
  "Create token": "创建令牌",
  "Copy this token now — it will not be shown again.": "请立即复制这个令牌，关闭后将不再显示。",
  "Copy token": "复制令牌",
  "PUBLISHED VERSIONS": "已发布版本",
  "Untitled group": "未命名分组",
  "Untitled host": "未命名主机",
  "{count} entries": "{count} 个条目",
  "Connection details needed": "请完善连接信息",
  "Group": "分组",
  "New group": "新建分组",
  "Organize related connections": "组织相关连接",
  "Direct connection — no jump host configured.": "直接连接，未配置跳板机。",
  "Hop {number}": "第 {number} 个跳板",
  "Remove": "移除",
  "Bastion name": "跳板机名称",
  "No commands will be sent after login.": "登录后不会自动发送命令。",
  "Command {number}": "第 {number} 条命令",
  "Unsaved changes": "有未保存的更改",
  "Hide": "隐藏",
  "Delete \"{name}\"?": "确定删除“{name}”吗？",
  "this entry": "这个条目",
  "Valid": "有效",
  "Needs attention": "需要完善",
  "Ready to save and publish.": "配置有效，可以保存并发布。",
  "Configuration is incomplete.": "配置尚未填写完整。",
  "Unavailable": "暂不可用",
  "Fix the validation error before saving.": "请先修复配置错误再保存。",
  "Draft saved.": "草稿已保存。",
  "Version {version} published.": "版本 {version} 已发布。",
  "No device tokens yet.": "还没有设备令牌。",
  "Revoked {date}": "已于 {date} 吊销",
  "Created {date}": "创建于 {date}",
  "Revoke": "吊销",
  "Revoked": "已吊销",
  "Enter a device name.": "请输入设备名称。",
  "Device token created.": "设备令牌已创建。",
  "Revoke this device token?": "确定吊销这个设备令牌吗？",
  "Device access revoked.": "设备访问权限已吊销。",
  "Token copied.": "令牌已复制。",
  "No published versions yet.": "还没有已发布版本。",
  "Published configuration": "已发布配置",
  "Restore draft": "恢复为草稿",
  "Restore version {version} into the current draft?": "确定将版本 {version} 恢复为当前草稿吗？",
  "Version {version} restored to the draft.": "版本 {version} 已恢复为草稿。",
  "Request failed ({status})": "请求失败（{status}）",
  "Delay ms": "延迟（毫秒）",
  "Show secrets": "显示敏感信息",
  "Hide secrets": "隐藏敏感信息",
  "The selected group is no longer available.": "所选分组已不存在，请重新选择。",
};

let currentLanguage = localStorage.getItem("sshw-language") === "en" ? "en" : "zh-CN";

function t(source, values = {}) {
  let result = currentLanguage === "zh-CN" ? (translations[source] || source) : source;
  for (const [key, value] of Object.entries(values)) {
    result = result.replaceAll(`{${key}}`, value);
  }
  return result;
}

function applyLanguage() {
  document.documentElement.lang = currentLanguage;
  document.title = currentLanguage === "zh-CN" ? "sshw 配置中心" : "sshw Config Center";

  const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
  while (walker.nextNode()) {
    const node = walker.currentNode;
    if (["SCRIPT", "STYLE", "CODE"].includes(node.parentElement?.tagName)) continue;
    node._sshwSource ??= node.nodeValue;
    const source = node._sshwSource;
    const trimmed = source.trim();
    if (!trimmed || !translations[trimmed]) {
      if (currentLanguage === "en") node.nodeValue = source;
      continue;
    }
    node.nodeValue = source.replace(trimmed, t(trimmed));
  }

  for (const element of $$("[placeholder], [title], [aria-label]")) {
    for (const attribute of ["placeholder", "title", "aria-label"]) {
      if (!element.hasAttribute(attribute)) continue;
      const sourceKey = `sshw${attribute.replace("-", "")}Source`;
      element.dataset[sourceKey] ??= element.getAttribute(attribute);
      const source = element.dataset[sourceKey];
      element.setAttribute(attribute, translateAttribute(source));
    }
  }

  $$(".language-toggle").forEach((button) => {
    button.textContent = currentLanguage === "zh-CN" ? "EN" : "中文";
    button.setAttribute("aria-label", currentLanguage === "zh-CN" ? "切换到英文" : "Switch to Chinese");
  });
  const previewSecretsButton = $("#togglePreviewSecrets");
  if (previewSecretsButton) {
    previewSecretsButton.textContent = t(state.showPreviewSecrets ? "Hide secrets" : "Show secrets");
  }
}

function translateAttribute(source) {
  if (currentLanguage !== "zh-CN") return source;
  const attributes = {
    "Switch language": "切换语言",
    "Sign out": "退出登录",
    "Host configuration": "主机配置",
    "Move up": "上移",
    "Move down": "下移",
    "Delete": "删除",
    "Close": "关闭",
    "Production API": "生产环境 API",
    "Added production hosts": "新增生产环境主机",
    "Office MacBook": "办公电脑",
    "Choose a configuration file": "选择配置文件",
  };
  return attributes[source] || translations[source] || source;
}

function translateError(message) {
  if (currentLanguage !== "zh-CN" || !message) return message;
  const exact = {
    "configuration is empty": "配置为空",
    "configuration contains no hosts": "配置中没有主机",
    "invalid username or password": "用户名或密码错误",
    "authentication required": "请先登录",
    "too many login attempts": "登录尝试过于频繁，请稍后再试",
    "internal server error": "服务器内部错误",
    "version not found": "未找到该版本",
    "active token not found": "未找到有效令牌",
  };
  if (exact[message]) return exact[message];

  return message
    .replace(/^item (\d+)/, "第 $1 项")
    .replace(/: name is required/g, "：必须填写名称")
    .replace(/: host is required/g, "：必须填写主机地址")
    .replace(/port must be between 1 and 65535/g, "端口必须在 1 到 65535 之间")
    .replace(/configuration contains no hosts/g, "配置中没有主机")
    .replace(/group must contain at least one host or group/g, "分组中至少需要一个主机或子分组")
    .replace(/callback (\d+) requires a command/g, "第 $1 条登录后命令不能为空")
    .replace(/callback (\d+) delay cannot be negative/g, "第 $1 条登录后命令的延迟不能为负数")
    .replace(/draft is not publishable: /g, "草稿无法发布：")
    .replace(/invalid YAML: /g, "YAML 格式错误：")
    .replace(/configuration exceeds (\d+) bytes/g, "配置文件超过大小限制（$1 字节）")
    .replace(/invalid JSON: /g, "JSON 格式错误：");
}

function switchLanguage() {
  currentLanguage = currentLanguage === "zh-CN" ? "en" : "zh-CN";
  localStorage.setItem("sshw-language", currentLanguage === "en" ? "en" : "zh-CN");
  applyLanguage();
  renderAll();
  renderPreview();
  if ($("#devicesDialog").open) renderDevices();
  if ($("#historyDialog").open) renderHistory();
}

const state = {
  profile: "default",
  document: { nodes: [] },
  selectedPath: null,
  valid: false,
  dirty: false,
  yaml: "[]\n",
  stats: { hosts: 0, groups: 0, aliases: 0 },
  publishedVersion: 0,
  tokens: [],
  history: [],
  renderTimer: null,
  createKind: "host",
  createParentPath: null,
  importFile: null,
  showPreviewSecrets: false,
};

document.addEventListener("DOMContentLoaded", boot);

async function boot() {
  bindEvents();
  applyLanguage();
  try {
    await api("/api/session");
    await showApplication();
  } catch {
    showLogin();
  }
}

function bindEvents() {
  $$(".language-toggle").forEach((button) => button.addEventListener("click", switchLanguage));
  $("#loginForm").addEventListener("submit", login);
  $("#logoutButton").addEventListener("click", logout);
  $("#addHostButton").addEventListener("click", () => openCreateNode("host"));
  $("#emptyAddHostButton").addEventListener("click", () => openCreateNode("host"));
  $("#addGroupButton").addEventListener("click", () => openCreateNode("group"));
  $("#addChildHostButton").addEventListener("click", () => openCreateNode("host", state.selectedPath));
  $("#addChildGroupButton").addEventListener("click", () => openCreateNode("group", state.selectedPath));
  $("#createNodeForm").addEventListener("submit", createNodeFromForm);
  $("#cancelCreateNodeButton").addEventListener("click", () => $("#createNodeDialog").close());
  $("#importButton").addEventListener("click", openImportDialog);
  $("#importFileInput").addEventListener("change", (event) => selectImportFile(event.target.files[0]));
  $("#confirmImportButton").addEventListener("click", importConfiguration);
  $("#importDropZone").addEventListener("dragover", handleImportDragOver);
  $("#importDropZone").addEventListener("dragleave", () => $("#importDropZone").classList.remove("dragging"));
  $("#importDropZone").addEventListener("drop", handleImportDrop);
  $("#togglePreviewSecrets").addEventListener("click", togglePreviewSecrets);
  $("#deleteNodeButton").addEventListener("click", deleteSelected);
  $("#moveUpButton").addEventListener("click", () => moveSelected(-1));
  $("#moveDownButton").addEventListener("click", () => moveSelected(1));
  $("#saveButton").addEventListener("click", saveDraft);
  $("#publishButton").addEventListener("click", openPublish);
  $("#confirmPublishButton").addEventListener("click", publish);
  $("#devicesButton").addEventListener("click", () => {
    renderDevices();
    $("#devicesDialog").showModal();
  });
  $("#historyButton").addEventListener("click", () => {
    renderHistory();
    $("#historyDialog").showModal();
  });
  $("#createDeviceButton").addEventListener("click", createDeviceToken);
  $("#copyTokenButton").addEventListener("click", copyToken);
  $("#addJumpButton").addEventListener("click", addJump);
  $("#addCallbackButton").addEventListener("click", addCallback);
  $("#editorForm").addEventListener("submit", (event) => event.preventDefault());
  $("#editorForm").addEventListener("input", handleEditorInput);
  $("#editorForm").addEventListener("click", handleEditorClick);
  $("#hostTree").addEventListener("click", (event) => {
    const button = event.target.closest("[data-node-path]");
    if (!button) return;
    state.selectedPath = JSON.parse(button.dataset.nodePath);
    renderAll();
  });
  window.addEventListener("beforeunload", (event) => {
    if (!state.dirty) return;
    event.preventDefault();
    event.returnValue = "";
  });
}

async function login(event) {
  event.preventDefault();
  $("#loginError").textContent = "";
  try {
    await api("/api/login", {
      method: "POST",
      body: {
        username: $("#loginUsername").value.trim(),
        password: $("#loginPassword").value,
      },
    });
    $("#loginPassword").value = "";
    await showApplication();
  } catch (error) {
    $("#loginError").textContent = translateError(error.message);
  }
}

async function logout() {
  if (state.dirty && !window.confirm(t("Discard unsaved changes and sign out?"))) return;
  try {
    await api("/api/logout", { method: "POST", body: {} });
  } finally {
    showLogin();
  }
}

function showLogin() {
  $("#appView").classList.add("hidden");
  $("#loginView").classList.remove("hidden");
  setTimeout(() => $("#loginPassword").focus(), 50);
}

async function showApplication() {
  $("#loginView").classList.add("hidden");
  $("#appView").classList.remove("hidden");
  await loadState();
}

async function loadState() {
  const data = await api(`/api/admin/state?profile=${encodeURIComponent(state.profile)}`);
  state.document = data.document || { nodes: [] };
  state.document.nodes ||= [];
  state.yaml = data.yaml || "[]\n";
  state.publishedVersion = data.publishedVersion || 0;
  state.tokens = data.tokens || [];
  state.history = data.history || [];
  state.dirty = false;
  if (!pathExists(state.selectedPath)) {
    state.selectedPath = state.document.nodes.length ? [0] : null;
  }
  renderAll();
  schedulePreview(true);
}

function renderAll() {
  renderTree();
  renderEditor();
  renderMeta();
}

function renderTree() {
  const tree = $("#hostTree");
  tree.innerHTML = "";
  $("#treeEmpty").classList.toggle("hidden", state.document.nodes.length > 0);
  state.document.nodes.forEach((node, index) => {
    tree.appendChild(buildTreeNode(node, [index]));
  });
}

function buildTreeNode(node, path) {
  const wrapper = document.createElement("div");
  wrapper.className = "tree-node";
  const button = document.createElement("button");
  button.type = "button";
  button.className = "tree-item" + (samePath(path, state.selectedPath) ? " selected" : "");
  button.dataset.nodePath = JSON.stringify(path);

  const icon = document.createElement("span");
  icon.className = "tree-icon";
  icon.textContent = node.kind === "group" ? "▦" : "⌁";
  const copy = document.createElement("span");
  copy.className = "tree-copy";
  const name = document.createElement("strong");
  name.textContent = node.name || (node.kind === "group" ? t("Untitled group") : t("Untitled host"));
  const detail = document.createElement("span");
  detail.textContent = node.kind === "group"
    ? t("{count} entries", { count: (node.children || []).length })
    : [node.user, node.host].filter(Boolean).join("@") || t("Connection details needed");
  copy.append(name, detail);
  const alias = document.createElement("span");
  alias.className = "tree-alias";
  alias.textContent = node.alias || "";
  button.append(icon, copy, alias);
  wrapper.append(button);

  if (node.kind === "group" && (node.children || []).length) {
    const children = document.createElement("div");
    children.className = "tree-children";
    node.children.forEach((child, index) => {
      children.appendChild(buildTreeNode(child, [...path, index]));
    });
    wrapper.append(children);
  }
  return wrapper;
}

function renderEditor() {
  const node = getSelectedNode();
  $("#emptyEditor").classList.toggle("hidden", Boolean(node));
  $("#nodeEditor").classList.toggle("hidden", !node);
  if (!node) return;

  const group = node.kind === "group";
  $$(".host-only").forEach((element) => element.classList.toggle("hidden", group));
  $$(".group-only").forEach((element) => element.classList.toggle("hidden", !group));
  $("#nodeBreadcrumb").textContent = group ? t("Group") : t("Host");
  $("#editorTitle").textContent = node.name || (group ? t("New group") : t("New host"));
  $("#editorSubtitle").textContent = group ? t("Organize related connections") : t("Connection details and authentication");

  $$("[data-field]").forEach((input) => {
    const field = input.dataset.field;
    input.value = node[field] ?? "";
  });
  $$(".secret-field input").forEach((input) => input.type = "password");
  $$("[data-toggle-secret]").forEach((button) => button.textContent = t("Show"));
  $("#aliasCommand").textContent = `sshw ${node.alias || "prod-api"}`;

  const collection = getSelectedCollection();
  const index = state.selectedPath[state.selectedPath.length - 1];
  $("#moveUpButton").disabled = index === 0;
  $("#moveDownButton").disabled = !collection || index >= collection.length - 1;
  renderJumps(node);
  renderCallbacks(node);
}

function renderJumps(node) {
  const list = $("#jumpList");
  list.innerHTML = "";
  node.jump ||= [];
  if (!node.jump.length) {
    list.innerHTML = `<div class="nested-empty">${t("Direct connection — no jump host configured.")}</div>`;
    return;
  }
  node.jump.forEach((jump, index) => {
    const card = document.createElement("div");
    card.className = "nested-card";
    card.innerHTML = `
      <div class="nested-card-head"><span>${t("Hop {number}", { number: index + 1 })}</span><button type="button" data-remove-jump="${index}">${t("Remove")}</button></div>
      <div class="nested-fields">
        <input data-jump-index="${index}" data-jump-field="name" value="${escapeAttr(jump.name || "")}" placeholder="${t("Bastion name")}">
        <input data-jump-index="${index}" data-jump-field="host" value="${escapeAttr(jump.host || "")}" placeholder="bastion.example.com">
        <input data-jump-index="${index}" data-jump-field="port" value="${escapeAttr(jump.port || "")}" type="number" min="1" max="65535" placeholder="22">
        <input data-jump-index="${index}" data-jump-field="user" value="${escapeAttr(jump.user || "")}" placeholder="${t("User")}">
        <input data-jump-index="${index}" data-jump-field="keypath" value="${escapeAttr(jump.keypath || "")}" placeholder="${t("Private key path")}">
        <input data-jump-index="${index}" data-jump-field="password" value="${escapeAttr(jump.password || "")}" type="password" placeholder="${t("Password")}">
      </div>`;
    list.appendChild(card);
  });
}

function renderCallbacks(node) {
  const list = $("#callbackList");
  list.innerHTML = "";
  node["callback-shells"] ||= [];
  if (!node["callback-shells"].length) {
    list.innerHTML = `<div class="nested-empty">${t("No commands will be sent after login.")}</div>`;
    return;
  }
  node["callback-shells"].forEach((callback, index) => {
    const card = document.createElement("div");
    card.className = "nested-card";
    card.innerHTML = `
      <div class="nested-card-head"><span>${t("Command {number}", { number: index + 1 })}</span><button type="button" data-remove-callback="${index}">${t("Remove")}</button></div>
      <div class="nested-fields callback">
        <input data-callback-index="${index}" data-callback-field="cmd" value="${escapeAttr(callback.cmd || "")}" placeholder="cd /srv/application">
        <input data-callback-index="${index}" data-callback-field="delay" value="${escapeAttr(callback.delay || "")}" type="number" min="0" placeholder="${t("Delay ms")}">
      </div>`;
    list.appendChild(card);
  });
}

function renderMeta() {
  const counts = countNodes(state.document.nodes);
  $("#hostCount").textContent = counts.hosts;
  $("#deviceCount").textContent = state.tokens.filter((token) => !token.revokedAt).length;
  $("#versionCount").textContent = state.history.length;
  $("#profileName").textContent = state.profile;
  $("#draftState").textContent = state.dirty ? t("Unsaved changes") : t("Saved");
  $("#draftState").classList.toggle("dirty", state.dirty);
  $("#saveButton").disabled = !state.dirty || !state.valid;
  $("#publishButton").disabled = !state.valid;
}

function handleEditorInput(event) {
  const node = getSelectedNode();
  if (!node) return;
  if (event.target.dataset.field) {
    const field = event.target.dataset.field;
    node[field] = field === "port" ? numberOrZero(event.target.value) : event.target.value;
    if (field === "name") {
      $("#editorTitle").textContent = node.name || (node.kind === "group" ? t("New group") : t("New host"));
      renderTree();
    } else if (field === "alias") {
      $("#aliasCommand").textContent = `sshw ${node.alias || "prod-api"}`;
    }
  } else if (event.target.dataset.jumpIndex !== undefined) {
    const jump = node.jump[Number(event.target.dataset.jumpIndex)];
    const field = event.target.dataset.jumpField;
    jump[field] = field === "port" ? numberOrZero(event.target.value) : event.target.value;
  } else if (event.target.dataset.callbackIndex !== undefined) {
    const callback = node["callback-shells"][Number(event.target.dataset.callbackIndex)];
    const field = event.target.dataset.callbackField;
    callback[field] = field === "delay" ? numberOrZero(event.target.value) : event.target.value;
  }
  markDirty();
}

function handleEditorClick(event) {
  const toggle = event.target.closest("[data-toggle-secret]");
  if (toggle) {
    const input = $(`[data-field="${toggle.dataset.toggleSecret}"]`);
    const show = input.type === "password";
    input.type = show ? "text" : "password";
    toggle.textContent = show ? t("Hide") : t("Show");
    return;
  }
  if (event.target.dataset.removeJump !== undefined) {
    const node = getSelectedNode();
    node.jump.splice(Number(event.target.dataset.removeJump), 1);
    renderJumps(node);
    markDirty();
  }
  if (event.target.dataset.removeCallback !== undefined) {
    const node = getSelectedNode();
    node["callback-shells"].splice(Number(event.target.dataset.removeCallback), 1);
    renderCallbacks(node);
    markDirty();
  }
}

function openCreateNode(kind, parentPath = null) {
  state.createKind = kind;
  state.createParentPath = parentPath ? [...parentPath] : null;
  $("#createNodeForm").reset();
  $("#createNodePort").value = "22";
  $("#createNodeError").textContent = "";
  const nested = Boolean(parentPath);
  const title = kind === "group"
    ? (nested ? "Add nested group" : "Add group")
    : (nested ? "Add nested host" : "Add host");
  $("#createNodeTitle").textContent = t(title);
  $("#confirmCreateNodeButton").textContent = t(title);
  $$(".create-host-field").forEach((field) => field.classList.toggle("hidden", kind !== "host"));
  $("#createNodeDialog").showModal();
  setTimeout(() => $("#createNodeName").focus(), 30);
}

function createNodeFromForm(event) {
  event.preventDefault();
  const name = $("#createNodeName").value.trim();
  const host = $("#createNodeHost").value.trim();
  const port = numberOrZero($("#createNodePort").value);
  if (!name) {
    $("#createNodeError").textContent = t("Enter a display name.");
    $("#createNodeName").focus();
    return;
  }
  if (state.createKind === "host" && !host) {
    $("#createNodeError").textContent = t("Enter a hostname or IP.");
    $("#createNodeHost").focus();
    return;
  }
  if (state.createKind === "host" && (port < 1 || port > 65535)) {
    $("#createNodeError").textContent = t("Enter a valid port.");
    $("#createNodePort").focus();
    return;
  }

  const node = newNode(state.createKind);
  node.name = name;
  if (state.createKind === "host") {
    node.alias = $("#createNodeAlias").value.trim();
    node.host = host;
    node.user = $("#createNodeUser").value.trim();
    node.port = port;
  }

  let collection = state.document.nodes;
  let pathPrefix = [];
  if (state.createParentPath) {
    const parent = getNodeAtPath(state.createParentPath);
    if (!parent || parent.kind !== "group") {
      $("#createNodeError").textContent = t("The selected group is no longer available.");
      return;
    }
    parent.children ||= [];
    collection = parent.children;
    pathPrefix = state.createParentPath;
  }
  collection.push(node);
  state.selectedPath = [...pathPrefix, collection.length - 1];
  $("#createNodeDialog").close();
  markDirty();
  renderAll();
}

function newNode(kind) {
  if (kind === "group") return { kind: "group", name: "", children: [] };
  return { kind: "host", name: "", alias: "", host: "", user: "", port: 22, jump: [], "callback-shells": [] };
}

function openImportDialog() {
  state.importFile = null;
  $("#importFileInput").value = "";
  $("#importFileName").textContent = "";
  $("#importFileName").classList.add("hidden");
  $("#importError").textContent = "";
  $("#confirmImportButton").disabled = true;
  $("#importDialog").showModal();
}

function selectImportFile(file) {
  $("#importError").textContent = "";
  if (!file) {
    state.importFile = null;
    $("#confirmImportButton").disabled = true;
    return;
  }
  if (file.size > 10 * 1024 * 1024) {
    state.importFile = null;
    $("#importError").textContent = t("The selected file is larger than 10 MB.");
    $("#confirmImportButton").disabled = true;
    return;
  }
  state.importFile = file;
  $("#importFileName").textContent = `${file.name} · ${formatFileSize(file.size)}`;
  $("#importFileName").classList.remove("hidden");
  $("#confirmImportButton").disabled = false;
}

function handleImportDragOver(event) {
  event.preventDefault();
  $("#importDropZone").classList.add("dragging");
}

function handleImportDrop(event) {
  event.preventDefault();
  $("#importDropZone").classList.remove("dragging");
  selectImportFile(event.dataTransfer.files[0]);
}

async function importConfiguration() {
  if (!state.importFile) {
    $("#importError").textContent = t("Choose a file first.");
    return;
  }
  const button = $("#confirmImportButton");
  button.disabled = true;
  $("#importError").textContent = "";
  try {
    const result = await api("/api/admin/import", {
      method: "POST",
      rawBody: await state.importFile.text(),
      contentType: "application/yaml",
    });
    const mode = $('input[name="importMode"]:checked').value;
    if (mode === "replace" && state.document.nodes.length &&
        !window.confirm(t("Replace the current draft with this imported file?"))) {
      button.disabled = false;
      return;
    }
    const importedNodes = result.document?.nodes || [];
    state.document = mode === "append"
      ? { nodes: [...state.document.nodes, ...importedNodes] }
      : { nodes: importedNodes };
    state.selectedPath = importedNodes.length
      ? [mode === "append" ? state.document.nodes.length - importedNodes.length : 0]
      : null;
    $("#importDialog").close();
    markDirty();
    renderAll();
    await renderPreview();
    const message = t("Imported {hosts} hosts and {groups} groups. Review the draft, then save it.", {
      hosts: result.stats?.hosts || 0,
      groups: result.stats?.groups || 0,
    });
    toast(state.valid ? message : t("Imported entries need attention before they can be saved."), !state.valid);
  } catch (error) {
    $("#importError").textContent = translateError(error.message);
    button.disabled = false;
  }
}

function addJump() {
  const node = getSelectedNode();
  if (!node || node.kind !== "host") return;
  node.jump ||= [];
  node.jump.push({ kind: "host", name: "", host: "", user: "", port: 22 });
  renderJumps(node);
  markDirty();
}

function addCallback() {
  const node = getSelectedNode();
  if (!node || node.kind !== "host") return;
  node["callback-shells"] ||= [];
  node["callback-shells"].push({ cmd: "", delay: 0 });
  renderCallbacks(node);
  markDirty();
}

function deleteSelected() {
  const node = getSelectedNode();
  if (!node || !window.confirm(t('Delete "{name}"?', { name: node.name || t("this entry") }))) return;
  const parentPath = state.selectedPath.slice(0, -1);
  const collection = getSelectedCollection();
  const index = state.selectedPath[state.selectedPath.length - 1];
  collection.splice(index, 1);
  state.selectedPath = null;
  if (collection.length) {
    state.selectedPath = [...parentPath, Math.min(index, collection.length - 1)];
  } else if (parentPath.length) {
    state.selectedPath = parentPath;
  } else if (state.document.nodes.length) {
    state.selectedPath = [0];
  }
  markDirty();
  renderAll();
}

function moveSelected(offset) {
  const collection = getSelectedCollection();
  if (!collection) return;
  const index = state.selectedPath[state.selectedPath.length - 1];
  const next = index + offset;
  if (next < 0 || next >= collection.length) return;
  [collection[index], collection[next]] = [collection[next], collection[index]];
  state.selectedPath[state.selectedPath.length - 1] = next;
  markDirty();
  renderAll();
}

function markDirty() {
  state.dirty = true;
  renderMeta();
  schedulePreview();
}

function schedulePreview(immediate = false) {
  clearTimeout(state.renderTimer);
  if (immediate) return renderPreview();
  state.renderTimer = setTimeout(renderPreview, 220);
}

async function renderPreview() {
  try {
    const result = await api("/api/admin/render", {
      method: "POST",
      body: state.document,
    });
    state.yaml = result.yaml || "[]\n";
    state.valid = Boolean(result.valid);
    state.stats = result.stats || { hosts: 0, groups: 0, aliases: 0 };
    renderYAMLPreview();
    $("#validationBadge").textContent = state.valid ? t("Valid") : t("Needs attention");
    $("#validationBadge").className = `validation-badge ${state.valid ? "valid" : "invalid"}`;
    $("#validationMessage").textContent = state.valid
      ? t("Ready to save and publish.")
      : translateError(result.error || "Configuration is incomplete.");
    $("#validationMessage").classList.toggle("error", !state.valid);
    $("#statHosts").textContent = state.stats.hosts || 0;
    $("#statGroups").textContent = state.stats.groups || 0;
    $("#statAliases").textContent = state.stats.aliases || 0;
    renderMeta();
  } catch (error) {
    state.valid = false;
    $("#validationBadge").textContent = t("Unavailable");
    $("#validationBadge").className = "validation-badge invalid";
    $("#validationMessage").textContent = translateError(error.message);
    $("#validationMessage").classList.add("error");
    renderMeta();
  }
}

function togglePreviewSecrets() {
  state.showPreviewSecrets = !state.showPreviewSecrets;
  $("#togglePreviewSecrets").textContent = t(state.showPreviewSecrets ? "Hide secrets" : "Show secrets");
  renderYAMLPreview();
}

function renderYAMLPreview() {
  const yaml = state.showPreviewSecrets
    ? state.yaml
    : state.yaml.replace(/^(\s*(?:password|passphrase):).*$/gm, "$1 ••••••••");
  $("#yamlPreview").textContent = yaml;
}

async function saveDraft() {
  await renderPreview();
  if (!state.valid) {
    toast(t("Fix the validation error before saving."), true);
    return false;
  }
  try {
    const result = await api(`/api/admin/draft?profile=${encodeURIComponent(state.profile)}`, {
      method: "PUT",
      body: state.document,
    });
    state.dirty = false;
    state.yaml = result.yaml;
    renderMeta();
    toast(t("Draft saved."));
    return true;
  } catch (error) {
    toast(translateError(error.message), true);
    return false;
  }
}

async function openPublish() {
  if (state.dirty && !(await saveDraft())) return;
  $("#publishNote").value = "";
  $("#publishDialog").showModal();
}

async function publish(event) {
  event.preventDefault();
  try {
    const result = await api(`/api/admin/publish?profile=${encodeURIComponent(state.profile)}`, {
      method: "POST",
      body: { note: $("#publishNote").value.trim() },
    });
    $("#publishDialog").close();
    toast(t("Version {version} published.", { version: result.version }));
    await loadState();
  } catch (error) {
    toast(translateError(error.message), true);
  }
}

function renderDevices() {
  const list = $("#deviceList");
  list.innerHTML = "";
  $("#tokenReveal").classList.add("hidden");
  if (!state.tokens.length) {
    list.innerHTML = `<div class="list-empty">${t("No device tokens yet.")}</div>`;
    return;
  }
  state.tokens.forEach((token) => {
    const item = document.createElement("div");
    item.className = "device-item" + (token.revokedAt ? " revoked" : "");
    const copy = document.createElement("div");
    const name = document.createElement("strong");
    name.textContent = token.name;
    const detail = document.createElement("span");
    detail.textContent = token.revokedAt
      ? t("Revoked {date}", { date: formatDate(token.revokedAt) })
      : t("Created {date}", { date: formatDate(token.createdAt) });
    copy.append(name, detail);
    item.append(copy);
    if (!token.revokedAt) {
      const button = document.createElement("button");
      button.type = "button";
      button.textContent = t("Revoke");
      button.addEventListener("click", () => revokeToken(token.id));
      item.append(button);
    } else {
      const badge = document.createElement("span");
      badge.textContent = t("Revoked");
      item.append(badge);
    }
    list.append(item);
  });
}

async function createDeviceToken() {
  const name = $("#deviceName").value.trim();
  if (!name) {
    toast(t("Enter a device name."), true);
    return;
  }
  try {
    const result = await api(`/api/admin/tokens?profile=${encodeURIComponent(state.profile)}`, {
      method: "POST",
      body: { name },
    });
    $("#deviceName").value = "";
    $("#newToken").textContent = result.token;
    $("#tokenReveal").classList.remove("hidden");
    state.tokens.unshift(result.device);
    renderMeta();
    const list = $("#deviceList");
    renderDevices();
    $("#newToken").textContent = result.token;
    $("#tokenReveal").classList.remove("hidden");
    toast(t("Device token created."));
  } catch (error) {
    toast(translateError(error.message), true);
  }
}

async function revokeToken(id) {
  if (!window.confirm(t("Revoke this device token?"))) return;
  try {
    await api(`/api/admin/tokens/${id}?profile=${encodeURIComponent(state.profile)}`, { method: "DELETE" });
    await loadState();
    renderDevices();
    toast(t("Device access revoked."));
  } catch (error) {
    toast(translateError(error.message), true);
  }
}

async function copyToken() {
  const value = $("#newToken").textContent;
  if (!value) return;
  await navigator.clipboard.writeText(value);
  toast(t("Token copied."));
}

function renderHistory() {
  const list = $("#historyList");
  list.innerHTML = "";
  if (!state.history.length) {
    list.innerHTML = `<div class="list-empty">${t("No published versions yet.")}</div>`;
    return;
  }
  state.history.forEach((version) => {
    const item = document.createElement("div");
    item.className = "history-item";
    const copy = document.createElement("div");
    const note = document.createElement("strong");
    note.textContent = version.note || t("Published configuration");
    const detail = document.createElement("span");
    detail.textContent = `${formatDate(version.createdAt)} · ${version.sha256.slice(0, 12)}`;
    copy.append(note, detail);
    const number = document.createElement("span");
    const actions = document.createElement("div");
    actions.className = "history-actions";
    number.className = "history-version";
    number.textContent = `v${version.version}`;
    const restore = document.createElement("button");
    restore.type = "button";
    restore.textContent = t("Restore draft");
    restore.addEventListener("click", () => restoreVersion(version.version));
    actions.append(number, restore);
    item.append(copy, actions);
    list.append(item);
  });
}

async function restoreVersion(version) {
  if (!window.confirm(t("Restore version {version} into the current draft?", { version }))) return;
  try {
    await api(`/api/admin/restore?profile=${encodeURIComponent(state.profile)}`, {
      method: "POST",
      body: { version },
    });
    $("#historyDialog").close();
    await loadState();
    toast(t("Version {version} restored to the draft.", { version }));
  } catch (error) {
    toast(translateError(error.message), true);
  }
}

function getSelectedNode() {
  if (!state.selectedPath) return null;
  let nodes = state.document.nodes;
  let node = null;
  for (const index of state.selectedPath) {
    node = nodes[index];
    if (!node) return null;
    nodes = node.children || [];
  }
  return node;
}

function getNodeAtPath(path) {
  if (!path) return null;
  let nodes = state.document.nodes;
  let node = null;
  for (const index of path) {
    node = nodes[index];
    if (!node) return null;
    nodes = node.children || [];
  }
  return node;
}

function getSelectedCollection() {
  if (!state.selectedPath) return null;
  if (state.selectedPath.length === 1) return state.document.nodes;
  let node = state.document.nodes[state.selectedPath[0]];
  for (let i = 1; i < state.selectedPath.length - 1; i++) {
    node = (node.children || [])[state.selectedPath[i]];
    if (!node) return null;
  }
  return node.children || [];
}

function pathExists(path) {
  if (!path) return false;
  let nodes = state.document.nodes;
  for (const index of path) {
    if (!nodes[index]) return false;
    nodes = nodes[index].children || [];
  }
  return true;
}

function samePath(left, right) {
  return Boolean(left && right && left.length === right.length && left.every((value, index) => value === right[index]));
}

function countNodes(nodes) {
  const result = { hosts: 0, groups: 0 };
  for (const node of nodes || []) {
    if (node.kind === "group") {
      result.groups++;
      const child = countNodes(node.children || []);
      result.hosts += child.hosts;
      result.groups += child.groups;
    } else {
      result.hosts++;
    }
  }
  return result;
}

function numberOrZero(value) {
  const number = Number(value);
  return Number.isFinite(number) ? number : 0;
}

function focusName() {
  setTimeout(() => $('[data-field="name"]')?.focus(), 30);
}

function formatDate(value) {
  return new Intl.DateTimeFormat(currentLanguage === "zh-CN" ? "zh-CN" : "en", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

function formatFileSize(bytes) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

function escapeAttr(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll('"', "&quot;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
}

async function api(path, options = {}) {
  const request = {
    method: options.method || "GET",
    credentials: "same-origin",
    headers: { Accept: "application/json" },
  };
  if (options.body !== undefined) {
    request.headers["Content-Type"] = "application/json";
    request.body = JSON.stringify(options.body);
  } else if (options.rawBody !== undefined) {
    request.headers["Content-Type"] = options.contentType || "text/plain";
    request.body = options.rawBody;
  }
  const response = await fetch(path, request);
  if (response.status === 204) return null;
  const contentType = response.headers.get("Content-Type") || "";
  const body = contentType.includes("application/json") ? await response.json() : { error: await response.text() };
  if (!response.ok) {
    if (response.status === 401 && path !== "/api/login" && path !== "/api/session") showLogin();
    throw new Error(body.error || t("Request failed ({status})", { status: response.status }));
  }
  return body;
}

let toastTimer;
function toast(message, error = false) {
  const element = $("#toast");
  element.textContent = message;
  element.className = `toast show${error ? " error" : ""}`;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => element.classList.remove("show"), 2800);
}
