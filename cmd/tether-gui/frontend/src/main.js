import './style.css';
import {Terminal} from '@xterm/xterm';
import '@xterm/xterm/css/xterm.css';
import {FitAddon} from '@xterm/addon-fit';

import {
    ListConnections,
    AddConnection,
    UpdateConnection,
    DeleteConnection,
    BrowseIdentityFile,
    OpenSession,
    WriteToSession,
    ResizeSession,
    CloseSession,
    HasMasterPassword,
    CreateMasterPassword,
    UnlockMasterPassword,
    ListSnippets,
    AddSnippet,
    UpdateSnippet,
    DeleteSnippet,
} from '../wailsjs/go/main/App';
import {EventsOn, EventsOff, ClipboardGetText} from '../wailsjs/runtime/runtime';

const appEl = document.getElementById('app');
const connectionListEl = document.getElementById('connection-list');
const tabsContainerEl = document.getElementById('tabs-container');
const terminalAreaEl = document.getElementById('terminal-area');
const emptyStateEl = document.getElementById('empty-state');
const addBtn = document.getElementById('add-btn');
const addModal = document.getElementById('add-modal');
const addModalTitle = document.getElementById('add-modal-title');
const addForm = document.getElementById('add-form');
const addCancel = document.getElementById('add-cancel');
const addError = document.getElementById('add-error');
const browseIdentityBtn = document.getElementById('browse-identity-btn');
const clearPasswordRow = document.getElementById('clear-password-row');
const snippetsSectionEl = document.getElementById('snippets-section');
const snippetsListEl = document.getElementById('snippets-list');
const addSnippetBtn = document.getElementById('add-snippet-btn');
const snippetModal = document.getElementById('snippet-modal');
const snippetModalTitle = document.getElementById('snippet-modal-title');
const snippetForm = document.getElementById('snippet-form');
const snippetCancel = document.getElementById('snippet-cancel');
const snippetDelete = document.getElementById('snippet-delete');
const snippetError = document.getElementById('snippet-error');
const snippetCategoryOptions = document.getElementById('snippet-category-options');
const authOverlay = document.getElementById('auth-overlay');
const authTitle = document.getElementById('auth-title');
const authDescription = document.getElementById('auth-description');
const authConfirmRow = document.getElementById('auth-confirm-row');
const authForm = document.getElementById('auth-form');
const authError = document.getElementById('auth-error');

// Name of the connection currently being edited, or null when the modal is
// in "add" mode.
let editingName = null;

/** @type {Map<string, {name: string, term: Terminal, fit: FitAddon, pane: HTMLElement, tab: HTMLElement}>} */
const sessions = new Map();
let activeSessionId = null;

// Name/ID of the snippet currently being edited, or null when the modal is
// in "add" mode.
let editingSnippetId = null;

async function refreshSnippets() {
    const snippets = await ListSnippets();
    snippetsListEl.innerHTML = '';
    snippetCategoryOptions.innerHTML = '';

    if (!snippets || snippets.length === 0) {
        const li = document.createElement('li');
        li.className = 'empty-list';
        li.textContent = 'No snippets yet.';
        snippetsListEl.appendChild(li);
        return;
    }

    // Group by category, preserving first-seen category order.
    const byCategory = new Map();
    for (const s of snippets) {
        if (!byCategory.has(s.category)) byCategory.set(s.category, []);
        byCategory.get(s.category).push(s);
    }

    for (const category of byCategory.keys()) {
        const option = document.createElement('option');
        option.value = category;
        snippetCategoryOptions.appendChild(option);
    }

    for (const [category, items] of byCategory) {
        const header = document.createElement('li');
        header.className = 'snippet-category-header';
        header.textContent = category;
        snippetsListEl.appendChild(header);

        for (const s of items) {
            const li = document.createElement('li');
            li.className = 'snippet-item';
            li.title = 'Insert into the active session (not submitted)';

            const info = document.createElement('div');
            info.className = 'snippet-info';
            const label = document.createElement('span');
            label.className = 'snippet-label';
            label.textContent = s.label;
            const command = document.createElement('span');
            command.className = 'snippet-command';
            command.textContent = s.command;
            info.appendChild(label);
            info.appendChild(command);

            const edit = document.createElement('button');
            edit.className = 'snippet-edit';
            edit.type = 'button';
            edit.textContent = '✎';
            edit.title = 'Edit snippet';
            edit.addEventListener('click', (e) => {
                e.stopPropagation();
                openEditSnippetModal(s);
            });

            li.appendChild(info);
            li.appendChild(edit);
            li.addEventListener('click', () => insertSnippet(s.command));

            snippetsListEl.appendChild(li);
        }
    }
}

function insertSnippet(command) {
    snippetsSectionEl.open = false;

    if (!activeSessionId || !sessions.has(activeSessionId)) {
        alert('Open a session first, then click a snippet to insert it.');
        return;
    }
    // Route through xterm's own paste handling (term.paste), the same path
    // real clipboard paste uses, rather than writing straight to the
    // session. A raw WriteToSession call skips xterm's bracketed-paste
    // handling, and some remote shells (zsh/fish with autosuggestion
    // plugins) redraw the input line when they receive an unbracketed
    // burst of text, which showed up as the command appearing twice.
    // term.paste() still ends up calling WriteToSession under the hood via
    // the existing onData listener, just through the correct pipeline.
    // No trailing newline: this types the command into the terminal
    // without submitting it, so it can be edited (e.g. replacing
    // "mysession") before pressing Enter.
    const s = sessions.get(activeSessionId);
    s.term.paste(command);
    s.term.focus();
}

// Close the dropdown on an outside click, like a normal menu.
document.addEventListener('click', (e) => {
    if (snippetsSectionEl.open && !snippetsSectionEl.contains(e.target)) {
        snippetsSectionEl.open = false;
    }
});

addSnippetBtn.addEventListener('click', () => {
    snippetsSectionEl.open = false;
    editingSnippetId = null;
    snippetModalTitle.textContent = 'Add snippet';
    snippetError.classList.add('hidden');
    snippetDelete.classList.add('hidden');
    snippetForm.reset();
    snippetModal.classList.remove('hidden');
    snippetForm.category.focus();
});

function openEditSnippetModal(s) {
    snippetsSectionEl.open = false;
    editingSnippetId = s.id;
    snippetModalTitle.textContent = 'Edit snippet';
    snippetError.classList.add('hidden');
    snippetDelete.classList.remove('hidden');
    snippetForm.reset();
    snippetForm.category.value = s.category;
    snippetForm.label.value = s.label;
    snippetForm.command.value = s.command;
    snippetModal.classList.remove('hidden');
    snippetForm.category.focus();
}

snippetCancel.addEventListener('click', () => {
    snippetModal.classList.add('hidden');
});

snippetDelete.addEventListener('click', async () => {
    if (!editingSnippetId) return;
    try {
        await DeleteSnippet(editingSnippetId);
        snippetModal.classList.add('hidden');
        await refreshSnippets();
    } catch (err) {
        snippetError.textContent = String(err);
        snippetError.classList.remove('hidden');
    }
});

snippetForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    snippetError.classList.add('hidden');

    const input = {
        category: snippetForm.category.value.trim(),
        label: snippetForm.label.value.trim(),
        command: snippetForm.command.value.trim(),
    };

    try {
        if (editingSnippetId) {
            await UpdateSnippet(editingSnippetId, input);
        } else {
            await AddSnippet(input);
        }
        snippetModal.classList.add('hidden');
        await refreshSnippets();
    } catch (err) {
        snippetError.textContent = String(err);
        snippetError.classList.remove('hidden');
    }
});

async function refreshConnections() {
    const conns = await ListConnections();
    connectionListEl.innerHTML = '';

    if (!conns || conns.length === 0) {
        const li = document.createElement('li');
        li.className = 'empty-list';
        li.textContent = 'No saved connections yet.';
        connectionListEl.appendChild(li);
        return;
    }

    for (const c of conns) {
        const li = document.createElement('li');
        li.className = 'connection-item';

        const info = document.createElement('div');
        info.className = 'connection-info';
        const name = document.createElement('span');
        name.className = 'connection-name';
        name.textContent = c.name;
        const target = document.createElement('span');
        target.className = 'connection-target';
        target.textContent = `${c.user}@${c.host}:${c.port}`;
        info.appendChild(name);
        info.appendChild(target);

        const edit = document.createElement('button');
        edit.className = 'connection-edit';
        edit.textContent = '✎';
        edit.title = 'Edit connection';
        edit.addEventListener('click', (e) => {
            e.stopPropagation();
            openEditModal(c);
        });

        const del = document.createElement('button');
        del.className = 'connection-delete';
        del.textContent = '✕';
        del.title = 'Delete connection';
        del.addEventListener('click', async (e) => {
            e.stopPropagation();
            await DeleteConnection(c.name);
            await refreshConnections();
        });

        const actions = document.createElement('div');
        actions.className = 'connection-actions';
        actions.appendChild(edit);
        actions.appendChild(del);

        li.appendChild(info);
        li.appendChild(actions);
        li.addEventListener('click', () => openConnection(c.name));

        connectionListEl.appendChild(li);
    }
}

// xterm.js normally handles paste via the browser's native "paste" event
// landing on its hidden input element, but that doesn't fire reliably
// inside WebView2 (the GUI's embedded browser on Windows), and the
// standard browser clipboard API additionally needs permission grants
// that a WebView2 app doesn't have by default. Handle the common paste
// gestures explicitly via Wails' native clipboard binding instead: Ctrl+V,
// Shift+Insert, and right-click (PuTTY/Windows-terminal convention).
function attachPasteHandling(term, pane) {
    async function pasteFromClipboard() {
        try {
            const text = await ClipboardGetText();
            if (text) term.paste(text);
        } catch (err) {
            console.error('paste failed:', err);
        }
    }

    term.attachCustomKeyEventHandler((event) => {
        if (event.type !== 'keydown') return true;
        const isCtrlV = event.ctrlKey && !event.shiftKey && !event.altKey && event.key.toLowerCase() === 'v';
        const isShiftInsert = event.shiftKey && event.key === 'Insert';
        if (isCtrlV || isShiftInsert) {
            pasteFromClipboard();
            return false;
        }
        return true;
    });

    pane.addEventListener('contextmenu', (e) => {
        e.preventDefault();
        pasteFromClipboard();
    });
}

function updateEmptyState() {
    const hasSessions = sessions.size > 0;
    emptyStateEl.classList.toggle('hidden', hasSessions);
    terminalAreaEl.classList.toggle('hidden', !hasSessions);
}

async function openConnection(name) {
    let id;
    try {
        id = await OpenSession(name);
    } catch (err) {
        alert(`Failed to open session: ${err}`);
        return;
    }

    const pane = document.createElement('div');
    pane.className = 'terminal-pane';
    terminalAreaEl.appendChild(pane);

    const term = new Terminal({
        convertEol: true,
        theme: {background: '#12181f'},
        fontSize: 13,
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(pane);
    attachPasteHandling(term, pane);

    term.onData((data) => {
        WriteToSession(id, data).catch((err) => console.error(err));
    });
    term.onResize(({cols, rows}) => {
        ResizeSession(id, cols, rows).catch((err) => console.error(err));
    });

    const tab = document.createElement('div');
    tab.className = 'tab';
    const label = document.createElement('span');
    label.textContent = name;
    const closeBtn = document.createElement('button');
    closeBtn.className = 'tab-close';
    closeBtn.textContent = '✕';
    closeBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        // Remove the tab immediately rather than waiting for the backend's
        // "closed" event: a remote process can keep the pty's read pending
        // (e.g. a detached tmux session), which would otherwise leave the
        // tab stuck open even though the user asked to close it. The
        // CloseSession call still runs to tear down the ssh process.
        removeSession(id);
        CloseSession(id).catch((err) => console.error(err));
    });
    tab.appendChild(label);
    tab.appendChild(closeBtn);
    tab.addEventListener('click', () => activateSession(id));
    tabsContainerEl.appendChild(tab);

    // Re-fit after output arrives: the terminal's column count is computed
    // before xterm's own scrollbar appears, so once enough output triggers
    // scrollback the scrollbar starts overlapping the last column unless we
    // recompute. Throttled to at most once per animation frame.
    let fitScheduled = false;
    function scheduleFit() {
        if (fitScheduled) return;
        fitScheduled = true;
        requestAnimationFrame(() => {
            fitScheduled = false;
            // Fitting a hidden pane (display:none) would collapse it to
            // 0 cols/rows, so only refit while this tab is the visible one.
            if (id === activeSessionId) fit.fit();
        });
    }

    EventsOn(`session:${id}:data`, (chunk) => {
        term.write(chunk);
        scheduleFit();
    });
    EventsOn(`session:${id}:closed`, () => {
        removeSession(id);
    });

    sessions.set(id, {name, term, fit, pane, tab});
    activateSession(id);
    updateEmptyState();

    // Give the pane a layout pass before fitting.
    requestAnimationFrame(() => fit.fit());
}

function activateSession(id) {
    if (!sessions.has(id)) return;
    activeSessionId = id;
    for (const [sid, s] of sessions) {
        const isActive = sid === id;
        s.pane.classList.toggle('active', isActive);
        s.tab.classList.toggle('active', isActive);
    }
    const s = sessions.get(id);
    s.fit.fit();
    s.term.focus();
}

function removeSession(id) {
    const s = sessions.get(id);
    if (!s) return;

    EventsOff(`session:${id}:data`);
    EventsOff(`session:${id}:closed`);

    s.term.dispose();
    s.pane.remove();
    s.tab.remove();
    sessions.delete(id);

    if (activeSessionId === id) {
        activeSessionId = null;
        const next = sessions.keys().next().value;
        if (next) activateSession(next);
    }
    updateEmptyState();
}

window.addEventListener('resize', () => {
    if (activeSessionId && sessions.has(activeSessionId)) {
        sessions.get(activeSessionId).fit.fit();
    }
});

// Add/edit connection modal

browseIdentityBtn.addEventListener('click', async () => {
    try {
        const path = await BrowseIdentityFile();
        if (path) {
            addForm.identityFile.value = path;
        }
    } catch (err) {
        addError.textContent = String(err);
        addError.classList.remove('hidden');
    }
});

addBtn.addEventListener('click', () => {
    editingName = null;
    addModalTitle.textContent = 'Add connection';
    addError.classList.add('hidden');
    addForm.reset();
    addForm.port.value = 22;
    addForm.password.placeholder = '';
    clearPasswordRow.classList.add('hidden');
    addModal.classList.remove('hidden');
    addForm.name.focus();
});

function openEditModal(c) {
    editingName = c.name;
    addModalTitle.textContent = 'Edit connection';
    addError.classList.add('hidden');
    addForm.reset();
    addForm.name.value = c.name;
    addForm.host.value = c.host;
    addForm.user.value = c.user;
    addForm.port.value = c.port;
    addForm.identityFile.value = c.identityFile || c.identity_file || '';
    addForm.password.placeholder = c.hasPassword ? 'Leave blank to keep saved password' : 'No password saved';
    clearPasswordRow.classList.toggle('hidden', !c.hasPassword);
    addForm.clearPassword.checked = false;
    addModal.classList.remove('hidden');
    addForm.name.focus();
}

addCancel.addEventListener('click', () => {
    addModal.classList.add('hidden');
});

addForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    addError.classList.add('hidden');

    const input = {
        name: addForm.name.value.trim(),
        host: addForm.host.value.trim(),
        user: addForm.user.value.trim(),
        port: parseInt(addForm.port.value, 10) || 22,
        identityFile: addForm.identityFile.value.trim(),
        password: addForm.password.value,
        clearPassword: addForm.clearPassword.checked,
    };

    try {
        if (editingName) {
            await UpdateConnection(editingName, input);
        } else {
            await AddConnection(input);
        }
        addModal.classList.add('hidden');
        await refreshConnections();
    } catch (err) {
        addError.textContent = String(err);
        addError.classList.remove('hidden');
    }
});

// Master password gate. The vault key lives only in the Go backend's
// memory for this run; nothing about it is kept in the frontend beyond
// which mode (create vs. unlock) the overlay is currently showing.
let authMode = null; // 'create' | 'unlock'

async function startAuthFlow() {
    const hasMaster = await HasMasterPassword();
    authMode = hasMaster ? 'unlock' : 'create';

    if (authMode === 'create') {
        authTitle.textContent = 'Set a master password';
        authDescription.textContent = 'This protects any saved connection passwords and will be required every time you open Tether. It cannot be recovered if you lose it.';
        authConfirmRow.classList.remove('hidden');
        authForm.confirmPassword.required = true;
    } else {
        authTitle.textContent = 'Enter master password';
        authDescription.textContent = 'Enter your master password to continue.';
        authConfirmRow.classList.add('hidden');
        authForm.confirmPassword.required = false;
    }

    authError.classList.add('hidden');
    authForm.reset();
    authOverlay.classList.remove('hidden');
    authForm.password.focus();
}

authForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    authError.classList.add('hidden');

    const password = authForm.password.value;

    try {
        if (authMode === 'create') {
            if (password !== authForm.confirmPassword.value) {
                throw new Error('passwords do not match');
            }
            await CreateMasterPassword(password);
        } else {
            await UnlockMasterPassword(password);
        }
        authOverlay.classList.add('hidden');
        unlockApp();
    } catch (err) {
        authError.textContent = String(err);
        authError.classList.remove('hidden');
        authForm.password.value = '';
        authForm.password.focus();
    }
});

function unlockApp() {
    appEl.classList.remove('hidden');
    updateEmptyState();
    refreshConnections();
    refreshSnippets();
}

startAuthFlow();
