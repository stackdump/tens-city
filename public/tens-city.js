import { createClient } from 'https://cdn.jsdelivr.net/npm/@supabase/supabase-js/+esm';

// Canonical JSON stringification with sorted keys
// This ensures consistent JSON encoding across frontend and backend
function canonicalJSON(obj) {
    if (obj === null) {
        return 'null';
    }
    
    if (typeof obj !== 'object') {
        return JSON.stringify(obj);
    }
    
    if (Array.isArray(obj)) {
        const items = obj.map(item => canonicalJSON(item));
        return '[' + items.join(',') + ']';
    }
    
    // Object: sort keys alphabetically
    const keys = Object.keys(obj).sort();
    const pairs = keys.map(key => {
        return JSON.stringify(key) + ':' + canonicalJSON(obj[key]);
    });
    return '{' + pairs.join(',') + '}';
}

class TensCity extends HTMLElement {
    constructor() {
        super();
        this._root = null;
        this._supabase = null;
        this._user = null;
        this._aceEditor = null;
        this._aceEditorContainer = null;
        this._loginContainer = null;
        this._appContainer = null;
        this._permalinkAnchor = null;
        this._helpContainer = null;
        this._menuOpen = false;
        this._pendingPermalinkData = null; // Store permalink data before authentication
        this._appShown = false; // Track whether app has been initialized to prevent race condition where _showApp() is called multiple times
        this._ownershipCache = {}; // Cache ownership checks to avoid redundant API calls
        this._editorMode = 'jsonld'; // 'jsonld' or 'markdown'
        this._markdownEditor = null;
        this._markdownPreview = null;
        this._frontmatterForm = null;
        this._docTags = [];
        this._lastSavedCid = null; // Store CID from last save for permalink in markdown mode
        this._lastSavedSlug = null; // Store slug from last save for permalink in markdown mode
    }

    connectedCallback() {
        if (this._root) return;
        
        // Capture permalink data before authentication to preserve it across login flow
        this._capturePermalinkData();
        
        this._buildRoot();
        this._initSupabase();
        this._checkAuth();
    }

    _capturePermalinkData() {
        // Extract permalink data from URL before authentication
        // This ensures we don't lose the data when redirecting to login
        const urlParams = new URLSearchParams(window.location.search);
        const encodedData = urlParams.get('data');
        
        if (encodedData) {
            console.log('Permalink: Captured data from URL parameter');
            const permalinkData = this._decodePermalinkData(encodedData);
            
            if (permalinkData) {
                // Store in both instance variable and sessionStorage
                // sessionStorage persists across OAuth redirects
                this._pendingPermalinkData = permalinkData;
                sessionStorage.setItem('pendingPermalinkData', JSON.stringify(permalinkData));
                console.log('Permalink: Successfully parsed and stored permalink data (in memory and sessionStorage)');
            } else {
                this._pendingPermalinkData = null;
                sessionStorage.removeItem('pendingPermalinkData');
            }
        } else {
            // Check if we have pending data from sessionStorage (after OAuth redirect)
            const storedData = sessionStorage.getItem('pendingPermalinkData');
            if (storedData) {
                console.log('Permalink: Restored data from sessionStorage (after OAuth redirect)');
                try {
                    this._pendingPermalinkData = JSON.parse(storedData);
                } catch (err) {
                    console.error('Permalink: Failed to parse stored data:', err);
                    this._pendingPermalinkData = null;
                    sessionStorage.removeItem('pendingPermalinkData');
                }
            }
        }
    }

    _decodePermalinkData(encodedData) {
        // Utility method to decode URL-encoded JSON data
        // Handles multiple levels of encoding
        if (!encodedData) return null;
        
        try {
            // Decode recursively in case of double (or multiple) encoding
            let decodedData = encodedData;
            
            // Keep decoding until we can't decode anymore or get valid JSON
            while (true) {
                try {
                    const nextDecoded = decodeURIComponent(decodedData);
                    // If decoding doesn't change the string, we're done
                    if (nextDecoded === decodedData) {
                        break;
                    }
                    decodedData = nextDecoded;
                    
                    // Try to parse as JSON - if successful, we're done
                    JSON.parse(decodedData);
                    break;
                } catch (jsonErr) {
                    // Not valid JSON yet, continue decoding if possible
                    // The loop will continue and nextDecoded === decodedData will eventually be true
                }
            }
            
            const data = JSON.parse(decodedData);
            return {
                jsonString: JSON.stringify(data, null, 2),
                data: data
            };
        } catch (err) {
            console.error('Permalink: Failed to parse URL data:', err);
            return null;
        }
    }

    _buildRoot() {
        this._root = document.createElement('div');
        this._root.className = 'tc-root';
        this._applyStyles(this._root, {
            width: '100%',
            height: '100vh',
            display: 'flex',
            flexDirection: 'column',
            fontFamily: 'system-ui, -apple-system, "Segoe UI", Roboto, sans-serif',
            background: '#f5f5f5'
        });
        this.appendChild(this._root);

        // Login container (shown when not authenticated)
        this._loginContainer = document.createElement('div');
        this._loginContainer.className = 'tc-login';
        this._applyStyles(this._loginContainer, {
            display: 'none',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            height: '100%',
            gap: '20px'
        });
        this._root.appendChild(this._loginContainer);

        // App container (shown when authenticated)
        this._appContainer = document.createElement('div');
        this._appContainer.className = 'tc-app';
        this._applyStyles(this._appContainer, {
            display: 'none',
            flexDirection: 'column',
            height: '100%'
        });
        this._root.appendChild(this._appContainer);
    }

    _initSupabase() {
        // Get Supabase URL and anon key from attributes or use defaults for local development
        const supabaseUrl = this.getAttribute('supabase-url') || 'http://localhost:54321';
        const supabaseKey = this.getAttribute('supabase-key') || 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZS1kZW1vIiwicm9sZSI6ImFub24iLCJleHAiOjE5ODM4MTI5OTZ9.CRXP1A7WOeoJeXxjNni43kdQwgnWNReilDMblYTn_I0';
        
        try {
            this._supabase = createClient(supabaseUrl, supabaseKey);

            // Listen for auth state changes
            this._supabase.auth.onAuthStateChange(async (event, session) => {
                console.log('Auth state changed:', event, session);
                if (session?.user) {
                    this._user = session.user;
                    // Force app rebuild to ensure delete button is created with correct auth state
                    this._appShown = false;
                    await this._showApp();
                } else {
                    this._user = null;
                    this._showLogin();
                }
            });
        } catch (err) {
            console.error('Failed to initialize Supabase:', err);
            this._showError('Failed to initialize Supabase. Please check your configuration.');
        }
    }

    async _checkAuth() {
        if (!this._supabase) {
            this._showError('Supabase client not initialized. Please configure supabase-url and supabase-key attributes.');
            return;
        }
        try {
            const { data: { session } } = await this._supabase.auth.getSession();
            if (session?.user) {
                this._user = session.user;
                await this._showApp();
            } else {
                this._showLogin();
            }
        } catch (err) {
            console.error('Auth check failed:', err);
            this._showError('Failed to check authentication. Please check your Supabase configuration.');
        }
    }

    _showError(message) {
        this._loginContainer.style.display = 'none';
        this._appContainer.style.display = 'flex';
        this._appContainer.innerHTML = '';
        
        const errorContainer = document.createElement('div');
        this._applyStyles(errorContainer, {
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            height: '100%',
            padding: '24px',
            gap: '16px'
        });

        const errorIcon = document.createElement('div');
        errorIcon.textContent = '⚠️';
        this._applyStyles(errorIcon, {
            fontSize: '64px'
        });
        errorContainer.appendChild(errorIcon);

        const errorTitle = document.createElement('h2');
        errorTitle.textContent = 'Configuration Error';
        this._applyStyles(errorTitle, {
            fontSize: '24px',
            fontWeight: 'bold',
            margin: '0',
            color: '#d73a49'
        });
        errorContainer.appendChild(errorTitle);

        const errorMessage = document.createElement('p');
        errorMessage.textContent = message;
        this._applyStyles(errorMessage, {
            fontSize: '16px',
            color: '#586069',
            textAlign: 'center',
            maxWidth: '600px',
            margin: '0'
        });
        errorContainer.appendChild(errorMessage);

        const helpText = document.createElement('p');
        this._applyStyles(helpText, {
            fontSize: '14px',
            color: '#586069',
            textAlign: 'center',
            maxWidth: '600px',
            margin: '16px 0 0 0'
        });
        
        // Build help text safely without innerHTML
        helpText.textContent = 'Please configure the ';
        const code1 = document.createElement('code');
        code1.textContent = 'supabase-url';
        helpText.appendChild(code1);
        helpText.appendChild(document.createTextNode(' and '));
        const code2 = document.createElement('code');
        code2.textContent = 'supabase-key';
        helpText.appendChild(code2);
        helpText.appendChild(document.createTextNode(' attributes on the <tens-city> element.'));
        helpText.appendChild(document.createElement('br'));
        helpText.appendChild(document.createTextNode('See '));
        const link = document.createElement('a');
        link.href = 'readme.md';
        link.target = '_blank';
        link.textContent = 'readme.md';
        helpText.appendChild(link);
        helpText.appendChild(document.createTextNode(' for setup instructions.'));
        errorContainer.appendChild(helpText);

        this._appContainer.appendChild(errorContainer);
    }

    _shouldShowEditor() {
        // Check if user is viewing specific content that should open the editor
        const urlParams = new URLSearchParams(window.location.search);
        const cidParam = urlParams.get('cid');
        const dataParam = urlParams.get('data');
        
        return !!(cidParam || dataParam || this._pendingPermalinkData);
    }

    _showLogin() {
        this._appShown = false; // Reset app shown flag when showing login
        // Don't show full login screen - just show app with login button in header
        this._loginContainer.style.display = 'none';
        this._appContainer.style.display = 'flex';
        this._appContainer.innerHTML = '';
        
        this._createHeaderWithLogin();
        this._createToolbar();
        this._createEditor().then(() => {
            if (this._shouldShowEditor()) {
                // User is viewing specific content, show editor with that content
                this._showEditorContainer();
                this._loadInitialData();
            } else {
                // No specific content to view, redirect to latest post
                this._showLatestPostView();
            }
        });
    }

    async _loginWithGitHub() {
        try {
            const { error } = await this._supabase.auth.signInWithOAuth({
                provider: 'github',
                options: {
                    redirectTo: window.location.origin + window.location.pathname
                }
            });
            if (error) {
                console.error('Login error:', error);
                alert('Login failed: ' + error.message);
            }
        } catch (err) {
            console.error('Login exception:', err);
            alert('Login failed: ' + err.message);
        }
    }

    async _logout() {
        const { error } = await this._supabase.auth.signOut();
        if (error) {
            console.error('Logout error:', error);
            alert('Logout failed: ' + error.message);
        }
    }

    async _showApp() {
        // Prevent duplicate initialization from race condition between onAuthStateChange and _checkAuth completion
        // If app is already shown, just make it visible without rebuilding UI (preserves permalink data)
        if (this._appShown) {
            console.log('App already initialized, skipping duplicate _showApp() call');
            this._loginContainer.style.display = 'none';
            this._appContainer.style.display = 'flex';
            return;
        }
        
        this._appShown = true;
        this._loginContainer.style.display = 'none';
        this._appContainer.style.display = 'flex';
        this._appContainer.innerHTML = '';
        
        this._createHeader();
        this._createToolbar();
        await this._createEditor();
        
        if (this._shouldShowEditor()) {
            // User is viewing specific content, show editor with that content
            this._showEditorContainer();
            await this._loadInitialData();
        } else {
            // No specific content to view, redirect to latest post
            await this._showLatestPostView();
        }
    }

    _createHeader() {
        const header = document.createElement('div');
        header.className = 'tc-header';
        this._applyStyles(header, {
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            padding: '16px 24px',
            background: '#fff',
            borderBottom: '1px solid #e1e4e8',
            boxShadow: '0 1px 3px rgba(0,0,0,0.1)'
        });

        // Left section: hamburger menu + logo
        const leftSection = document.createElement('div');
        this._applyStyles(leftSection, {
            display: 'flex',
            alignItems: 'center',
            gap: '16px'
        });

        // Hamburger menu button
        const menuBtn = document.createElement('button');
        menuBtn.innerHTML = '☰';
        menuBtn.title = 'Menu';
        this._applyStyles(menuBtn, {
            padding: '8px 12px',
            fontSize: '24px',
            background: 'transparent',
            border: 'none',
            cursor: 'pointer',
            color: '#24292e',
            lineHeight: '1'
        });
        menuBtn.addEventListener('click', () => this._toggleMenu());
        leftSection.appendChild(menuBtn);

        // Logo
        const logo = document.createElement('img');
        logo.src = 'logo.svg';
        logo.alt = 'Tens City';
        logo.onerror = () => {
            // If logo fails to load, hide it gracefully
            logo.style.display = 'none';
        };
        this._applyStyles(logo, {
            height: '40px',
            width: 'auto'
        });
        leftSection.appendChild(logo);

        header.appendChild(leftSection);

        // Right section: empty (user info moved to menu)
        this._appContainer.appendChild(header);
    }

    _createHeaderWithLogin() {
        const header = document.createElement('div');
        header.className = 'tc-header';
        this._applyStyles(header, {
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            padding: '16px 24px',
            background: '#fff',
            borderBottom: '1px solid #e1e4e8',
            boxShadow: '0 1px 3px rgba(0,0,0,0.1)'
        });

        // Left section: hamburger menu + logo
        const leftSection = document.createElement('div');
        this._applyStyles(leftSection, {
            display: 'flex',
            alignItems: 'center',
            gap: '16px'
        });

        // Hamburger menu button
        const menuBtn = document.createElement('button');
        menuBtn.innerHTML = '☰';
        menuBtn.title = 'Menu';
        this._applyStyles(menuBtn, {
            padding: '8px 12px',
            fontSize: '24px',
            background: 'transparent',
            border: 'none',
            cursor: 'pointer',
            color: '#24292e',
            lineHeight: '1'
        });
        menuBtn.addEventListener('click', () => this._toggleMenu());
        leftSection.appendChild(menuBtn);

        // Logo
        const logo = document.createElement('img');
        logo.src = 'logo.svg';
        logo.alt = 'Tens City';
        logo.onerror = () => {
            // If logo fails to load, hide it gracefully
            logo.style.display = 'none';
        };
        this._applyStyles(logo, {
            height: '40px',
            width: 'auto'
        });
        leftSection.appendChild(logo);

        header.appendChild(leftSection);

        // Right section: empty (login button moved to menu)
        this._appContainer.appendChild(header);
    }

    _createToolbar() {
        const toolbar = document.createElement('div');
        toolbar.className = 'tc-toolbar';
        this._applyStyles(toolbar, {
            display: 'flex',
            gap: '8px',
            padding: '12px 24px',
            background: '#fff',
            borderBottom: '1px solid #e1e4e8'
        });

        const makeButton = (text, title, onClick) => {
            const btn = document.createElement('button');
            btn.textContent = text;
            btn.title = title;
            this._applyStyles(btn, {
                padding: '8px 16px',
                fontSize: '14px',
                fontWeight: '500',
                background: '#fafbfc',
                border: '1px solid #e1e4e8',
                borderRadius: '6px',
                cursor: 'pointer',
                transition: 'background 0.2s'
            });
            btn.addEventListener('mouseenter', () => {
                btn.style.background = '#f3f4f6';
            });
            btn.addEventListener('mouseleave', () => {
                btn.style.background = '#fafbfc';
            });
            btn.addEventListener('click', onClick);
            return btn;
        };

        // Editor mode toggle
        const modeToggle = makeButton(
            this._editorMode === 'jsonld' ? '📝 Switch to Markdown' : '📋 Switch to JSON-LD',
            'Toggle between JSON-LD and Markdown editors',
            () => this._toggleEditorMode()
        );
        modeToggle.id = 'tc-mode-toggle';
        toolbar.appendChild(modeToggle);

        // Add Save button only if user is authenticated
        if (this._user) {
            const saveBtn = makeButton('💾 Save', 'Save to create CID and update URL', () => this._saveData());
            saveBtn.id = 'tc-save-btn';
            toolbar.appendChild(saveBtn);
        }

        const clearBtn = makeButton('🗑️ Clear', 'Clear editor', () => this._clearEditor());

        // Add Delete button only if user is authenticated and viewing a CID
        if (this._user) {
            const deleteBtn = makeButton('🗑️ Delete', 'Delete this object (author only)', () => this._deleteObject());
            deleteBtn.style.display = 'none'; // Hidden by default, shown when viewing CID
            deleteBtn.id = 'tc-delete-btn';
            toolbar.appendChild(deleteBtn);
        }

        // Create permalink anchor styled as a button
        this._permalinkAnchor = document.createElement('a');
        this._permalinkAnchor.textContent = '🔗 Permalink';
        this._permalinkAnchor.title = 'Link to current data';
        this._permalinkAnchor.href = '#';
        this._permalinkAnchor.target = '_blank';
        this._applyStyles(this._permalinkAnchor, {
            padding: '8px 16px',
            fontSize: '14px',
            fontWeight: '500',
            background: '#fafbfc',
            border: '1px solid #e1e4e8',
            borderRadius: '6px',
            cursor: 'pointer',
            transition: 'background 0.2s',
            textDecoration: 'none',
            color: 'inherit',
            display: 'inline-block'
        });
        this._permalinkAnchor.addEventListener('mouseenter', () => {
            this._permalinkAnchor.style.background = '#f3f4f6';
        });
        this._permalinkAnchor.addEventListener('mouseleave', () => {
            this._permalinkAnchor.style.background = '#fafbfc';
        });

        toolbar.appendChild(clearBtn);
        toolbar.appendChild(this._permalinkAnchor);

        this._appContainer.appendChild(toolbar);
    }

    async _createEditor() {
        if (this._editorMode === 'jsonld') {
            await this._createJSONLDEditor();
        } else {
            await this._createMarkdownEditor();
        }
        
        // Update permalink after creating editor
        this._updatePermalinkAnchor();
        
        // Initially hide editor - it will be shown after redirect determination
        const editorContainer = this._appContainer.querySelector('.tc-editor-container');
        if (editorContainer) {
            editorContainer.style.display = 'none';
        }
    }

    _showEditorContainer() {
        // Show the editor container
        const editorContainer = this._appContainer.querySelector('.tc-editor-container');
        if (editorContainer) {
            editorContainer.style.display = 'flex';
        }
    }

    async _createJSONLDEditor() {
        const editorContainer = document.createElement('div');
        editorContainer.className = 'tc-editor-container';
        this._applyStyles(editorContainer, {
            flex: '1 1 auto',
            display: 'flex',
            flexDirection: 'column',
            padding: '24px',
            overflow: 'hidden',
            gap: '12px'
        });

        // Document preview panel (initially hidden)
        const previewPanel = document.createElement('div');
        previewPanel.className = 'tc-doc-preview-panel';
        previewPanel.id = 'tc-doc-preview-panel';
        this._applyStyles(previewPanel, {
            display: 'none',
            background: '#fff',
            borderRadius: '6px',
            border: '1px solid #e1e4e8',
            maxHeight: '300px',
            overflow: 'hidden',
            flexDirection: 'column',
            boxShadow: '0 1px 3px rgba(0,0,0,0.1)'
        });

        const previewHeader = document.createElement('div');
        this._applyStyles(previewHeader, {
            padding: '8px 16px',
            background: '#f5f5f5',
            borderBottom: '1px solid #e1e4e8',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            fontWeight: '600',
            fontSize: '14px'
        });

        const previewTitle = document.createElement('span');
        previewTitle.textContent = 'Linked Documents';
        previewHeader.appendChild(previewTitle);

        const closePreviewBtn = document.createElement('button');
        closePreviewBtn.textContent = '×';
        this._applyStyles(closePreviewBtn, {
            background: 'transparent',
            border: 'none',
            fontSize: '24px',
            cursor: 'pointer',
            padding: '0',
            lineHeight: '1',
            color: '#586069'
        });
        closePreviewBtn.addEventListener('click', () => {
            previewPanel.style.display = 'none';
        });
        previewHeader.appendChild(closePreviewBtn);

        previewPanel.appendChild(previewHeader);

        const previewContent = document.createElement('div');
        previewContent.id = 'tc-doc-preview-content';
        this._applyStyles(previewContent, {
            padding: '16px',
            overflowY: 'auto',
            flex: '1'
        });
        previewPanel.appendChild(previewContent);

        editorContainer.appendChild(previewPanel);

        // Editor wrapper
        const editorWrapper = document.createElement('div');
        editorWrapper.className = 'tc-editor-wrapper';
        this._applyStyles(editorWrapper, {
            flex: '1',
            background: '#fff',
            borderRadius: '6px',
            border: '1px solid #e1e4e8',
            overflow: 'hidden',
            display: 'flex',
            flexDirection: 'column',
            boxShadow: '0 1px 3px rgba(0,0,0,0.1)'
        });

        const editorDiv = document.createElement('div');
        editorDiv.className = 'tc-editor';
        this._applyStyles(editorDiv, {
            width: '100%',
            height: '100%',
            minHeight: '400px'
        });

        editorWrapper.appendChild(editorDiv);
        editorContainer.appendChild(editorWrapper);
        this._appContainer.appendChild(editorContainer);

        // Initialize ACE editor
        await this._initAceEditor(editorDiv);
    }

    async _createMarkdownEditor() {
        // Load marked library for markdown preview
        await this._loadScript('https://cdn.jsdelivr.net/npm/marked@11.0.0/marked.min.js', 'marked');
        // Load DOMPurify for HTML sanitization
        await this._loadScript('https://cdn.jsdelivr.net/npm/dompurify@3.0.6/dist/purify.min.js', 'DOMPurify');

        const editorContainer = document.createElement('div');
        editorContainer.className = 'tc-editor-container';
        this._applyStyles(editorContainer, {
            flex: '1 1 auto',
            display: 'flex',
            flexDirection: 'row',
            gap: '24px',
            padding: '24px',
            overflow: 'hidden'
        });

        // Left side: Markdown editor and preview
        const leftPane = document.createElement('div');
        this._applyStyles(leftPane, {
            flex: '1 1 60%',
            display: 'flex',
            flexDirection: 'column',
            gap: '12px',
            overflow: 'hidden'
        });

        // Markdown editor
        const editorWrapper = document.createElement('div');
        this._applyStyles(editorWrapper, {
            flex: '1 1 50%',
            background: '#fff',
            borderRadius: '6px',
            border: '1px solid #e1e4e8',
            overflow: 'hidden',
            display: 'flex',
            flexDirection: 'column',
            boxShadow: '0 1px 3px rgba(0,0,0,0.1)'
        });

        const editorHeader = document.createElement('div');
        editorHeader.textContent = 'Markdown Content';
        this._applyStyles(editorHeader, {
            padding: '8px 16px',
            background: '#f5f5f5',
            borderBottom: '1px solid #e1e4e8',
            fontWeight: '600',
            fontSize: '14px'
        });
        editorWrapper.appendChild(editorHeader);

        this._markdownEditor = document.createElement('textarea');
        this._applyStyles(this._markdownEditor, {
            flex: '1',
            width: '100%',
            border: 'none',
            resize: 'none',
            fontFamily: 'Monaco, Menlo, "Ubuntu Mono", monospace',
            fontSize: '14px',
            padding: '16px'
        });
        this._markdownEditor.placeholder = 'Write your markdown content here...';
        this._markdownEditor.addEventListener('input', () => this._updateMarkdownPreview());
        editorWrapper.appendChild(this._markdownEditor);

        leftPane.appendChild(editorWrapper);

        // Preview pane
        const previewWrapper = document.createElement('div');
        this._applyStyles(previewWrapper, {
            flex: '1 1 50%',
            background: '#fff',
            borderRadius: '6px',
            border: '1px solid #e1e4e8',
            overflow: 'hidden',
            display: 'flex',
            flexDirection: 'column',
            boxShadow: '0 1px 3px rgba(0,0,0,0.1)'
        });

        const previewHeader = document.createElement('div');
        previewHeader.textContent = 'Preview';
        this._applyStyles(previewHeader, {
            padding: '8px 16px',
            background: '#f5f5f5',
            borderBottom: '1px solid #e1e4e8',
            fontWeight: '600',
            fontSize: '14px'
        });
        previewWrapper.appendChild(previewHeader);

        this._markdownPreview = document.createElement('div');
        this._applyStyles(this._markdownPreview, {
            flex: '1',
            padding: '16px',
            overflowY: 'auto',
            lineHeight: '1.6'
        });
        this._markdownPreview.innerHTML = '<p style="color: #999;">Preview will appear here...</p>';
        previewWrapper.appendChild(this._markdownPreview);

        leftPane.appendChild(previewWrapper);
        editorContainer.appendChild(leftPane);

        // Right side: Frontmatter form
        this._frontmatterForm = this._createFrontmatterForm();
        editorContainer.appendChild(this._frontmatterForm);

        this._appContainer.appendChild(editorContainer);
    }

    _createFrontmatterForm() {
        const form = document.createElement('div');
        form.className = 'tc-frontmatter-form';
        this._applyStyles(form, {
            flex: '0 0 300px',
            background: '#f5f5f5',
            borderRadius: '6px',
            border: '1px solid #e1e4e8',
            padding: '16px',
            overflowY: 'auto',
            boxShadow: '0 1px 3px rgba(0,0,0,0.1)'
        });

        const title = document.createElement('h3');
        title.textContent = 'Document Metadata';
        this._applyStyles(title, {
            marginTop: '0',
            marginBottom: '16px',
            fontSize: '16px',
            fontWeight: 'bold'
        });
        form.appendChild(title);

        // Helper to create form fields
        const createField = (label, id, type = 'text', required = false, placeholder = '') => {
            const group = document.createElement('div');
            this._applyStyles(group, {
                marginBottom: '12px'
            });

            const labelEl = document.createElement('label');
            labelEl.textContent = label + (required ? ' *' : '');
            labelEl.htmlFor = id;
            this._applyStyles(labelEl, {
                display: 'block',
                marginBottom: '4px',
                fontSize: '13px',
                fontWeight: '600'
            });
            group.appendChild(labelEl);

            let input;
            if (type === 'textarea') {
                input = document.createElement('textarea');
                this._applyStyles(input, {
                    minHeight: '60px',
                    resize: 'vertical'
                });
            } else {
                input = document.createElement('input');
                input.type = type;
            }
            
            input.id = id;
            input.placeholder = placeholder;
            if (required) input.required = true;
            this._applyStyles(input, {
                width: '100%',
                padding: '6px 8px',
                border: '1px solid #e1e4e8',
                borderRadius: '4px',
                fontSize: '13px',
                fontFamily: 'inherit'
            });
            group.appendChild(input);

            return group;
        };

        // Author info (read-only display)
        if (this._user) {
            const authorGroup = document.createElement('div');
            this._applyStyles(authorGroup, {
                marginBottom: '16px',
                padding: '12px',
                background: '#fff',
                borderRadius: '4px',
                border: '1px solid #d1d5da'
            });

            const authorLabel = document.createElement('div');
            authorLabel.textContent = 'Author';
            this._applyStyles(authorLabel, {
                fontSize: '13px',
                fontWeight: '600',
                marginBottom: '8px',
                color: '#24292e'
            });
            authorGroup.appendChild(authorLabel);

            const authorInfo = document.createElement('div');
            this._applyStyles(authorInfo, {
                display: 'flex',
                alignItems: 'center',
                gap: '8px'
            });

            // GitHub icon
            const githubIcon = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
            githubIcon.setAttribute('height', '16');
            githubIcon.setAttribute('width', '16');
            githubIcon.setAttribute('viewBox', '0 0 16 16');
            githubIcon.setAttribute('fill', 'currentColor');
            const githubPath = document.createElementNS('http://www.w3.org/2000/svg', 'path');
            githubPath.setAttribute('d', 'M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z');
            githubIcon.appendChild(githubPath);
            authorInfo.appendChild(githubIcon);

            const authorName = document.createElement('span');
            authorName.textContent = this._user.user_metadata?.user_name || this._user.email || 'User';
            this._applyStyles(authorName, {
                fontSize: '13px',
                color: '#24292e',
                fontWeight: '500'
            });
            authorInfo.appendChild(authorName);

            authorGroup.appendChild(authorInfo);

            const authorNote = document.createElement('div');
            authorNote.textContent = 'Author is set from your GitHub login';
            this._applyStyles(authorNote, {
                fontSize: '11px',
                color: '#586069',
                marginTop: '6px',
                fontStyle: 'italic'
            });
            authorGroup.appendChild(authorNote);

            form.appendChild(authorGroup);
        }

        form.appendChild(createField('Title', 'fm-title', 'text', true, 'Document title'));
        form.appendChild(createField('Description', 'fm-description', 'textarea', false, 'Short description'));
        form.appendChild(createField('Slug', 'fm-slug', 'text', false, 'url-friendly-slug'));
        form.appendChild(createField('Date Published', 'fm-datePublished', 'datetime-local', true));
        form.appendChild(createField('Date Modified', 'fm-dateModified', 'datetime-local', false));

        // Set default date
        const now = new Date().toISOString().slice(0, 16);
        form.querySelector('#fm-datePublished').value = now;

        // Auto-generate slug from title
        form.querySelector('#fm-title').addEventListener('input', (e) => {
            const slugField = form.querySelector('#fm-slug');
            if (!slugField.value) {
                slugField.value = this._generateSlug(e.target.value);
            }
        });

        return form;
    }

    _generateSlug(text) {
        return text
            .toLowerCase()
            .replace(/[^a-z0-9\s-]/g, '')
            .replace(/\s+/g, '-')
            .replace(/-+/g, '-')
            .replace(/^-|-$/g, '');
    }

    _updateMarkdownPreview() {
        if (!this._markdownEditor || !this._markdownPreview) return;
        
        const markdown = this._markdownEditor.value;
        if (markdown && window.marked && window.DOMPurify) {
            // Parse markdown and sanitize HTML output to prevent XSS
            const html = window.marked.parse(markdown, {
                breaks: true,
                gfm: true,
                headerIds: false,
                mangle: false
            });
            
            // Sanitize with DOMPurify before rendering
            this._markdownPreview.innerHTML = window.DOMPurify.sanitize(html);
        } else if (!markdown) {
            this._markdownPreview.innerHTML = '<p style="color: #999;">Preview will appear here...</p>';
        } else {
            this._markdownPreview.textContent = 'Loading preview...';
        }
    }

    async _initAceEditor(container) {
        const aceCdn = 'https://cdnjs.cloudflare.com/ajax/libs/ace/1.4.14/ace.js';
        try {
            await this._loadScript(aceCdn);
        } catch (err) {
            console.error('Failed to load ACE editor:', err);
            container.textContent = 'Failed to load editor';
            return;
        }

        const editor = window.ace.edit(container);
        editor.setTheme('ace/theme/textmate');
        editor.session.setMode('ace/mode/json');

        const opts = {
            fontSize: '14px',
            showPrintMargin: false,
            wrap: true,
            useWorker: true
        };

        editor.setOptions(opts);
        editor.session.setValue('{\n  "@context": "https://pflow.xyz/schema",\n  "@type": "Object"\n}');

        // Add change listener to update script tag and permalink anchor
        editor.session.on('change', () => {
            this._updateScriptTag();
            this._updatePermalinkAnchor();
            this._detectAndShowDocumentLinks();
        });

        this._aceEditor = editor;
        this._aceEditorContainer = container;
    }

    _loadScript(src, globalVar = 'ace') {
        return new Promise((resolve, reject) => {
            if (window[globalVar]) return resolve();
            if (document.querySelector(`script[src="${src}"]`)) {
                const check = () => window[globalVar] ? resolve() : setTimeout(check, 50);
                return check();
            }
            const s = document.createElement('script');
            s.src = src;
            s.onload = () => resolve();
            s.onerror = (e) => reject(e);
            document.head.appendChild(s);
        });
    }

    async _detectAndShowDocumentLinks() {
        // Only detect links in JSON-LD mode
        if (this._editorMode !== 'jsonld' || !this._aceEditor) return;

        const previewPanel = document.getElementById('tc-doc-preview-panel');
        const previewContent = document.getElementById('tc-doc-preview-content');
        
        if (!previewPanel || !previewContent) return;

        try {
            const editorContent = this._aceEditor.session.getValue();
            const data = JSON.parse(editorContent);

            // Extract URLs that might be links to our hosted documents
            const docLinks = this._extractDocumentLinks(data);

            if (docLinks.length === 0) {
                previewPanel.style.display = 'none';
                return;
            }

            // Show preview panel
            previewPanel.style.display = 'flex';

            // Load and display previews for each linked document
            previewContent.innerHTML = '<p style="color: #999; margin: 0;">Loading previews...</p>';

            const previews = await Promise.all(
                docLinks.map(link => this._loadDocumentPreview(link))
            );

            // Display previews
            previewContent.innerHTML = '';
            previews.forEach((preview, idx) => {
                if (preview) {
                    const previewItem = document.createElement('div');
                    this._applyStyles(previewItem, {
                        marginBottom: idx < previews.length - 1 ? '16px' : '0',
                        paddingBottom: idx < previews.length - 1 ? '16px' : '0',
                        borderBottom: idx < previews.length - 1 ? '1px solid #e1e4e8' : 'none'
                    });

                    const title = document.createElement('h4');
                    this._applyStyles(title, {
                        margin: '0 0 8px 0',
                        fontSize: '14px',
                        fontWeight: '600'
                    });

                    const link = document.createElement('a');
                    link.href = preview.url;
                    link.target = '_blank';
                    link.textContent = preview.title || preview.url;
                    this._applyStyles(link, {
                        color: '#0066cc',
                        textDecoration: 'none'
                    });
                    link.addEventListener('mouseenter', () => {
                        link.style.textDecoration = 'underline';
                    });
                    link.addEventListener('mouseleave', () => {
                        link.style.textDecoration = 'none';
                    });
                    title.appendChild(link);
                    previewItem.appendChild(title);

                    if (preview.description) {
                        const desc = document.createElement('p');
                        desc.textContent = preview.description;
                        this._applyStyles(desc, {
                            margin: '0',
                            fontSize: '13px',
                            color: '#586069',
                            lineHeight: '1.5'
                        });
                        previewItem.appendChild(desc);
                    }

                    previewContent.appendChild(previewItem);
                }
            });

            if (previewContent.innerHTML === '') {
                previewContent.innerHTML = '<p style="color: #999; margin: 0;">No previews available.</p>';
            }

        } catch (err) {
            // Invalid JSON or other error - hide preview panel
            previewPanel.style.display = 'none';
        }
    }

    _extractDocumentLinks(obj, links = []) {
        // Recursively extract URLs that might point to our hosted documents
        if (typeof obj !== 'object' || obj === null) return links;

        for (const key in obj) {
            const value = obj[key];
            
            // Check if this looks like a URL field
            if (typeof value === 'string' && (key === 'url' || key === '@id' || key.endsWith('Url'))) {
                // Check if it's a URL pointing to /posts/
                if (value.includes('/posts/') && !value.endsWith('.jsonld')) {
                    const slug = value.split('/posts/').pop().split(/[?#]/)[0];
                    if (slug && !links.includes(slug)) {
                        links.push(slug);
                    }
                }
            } else if (typeof value === 'object') {
                this._extractDocumentLinks(value, links);
            }
        }

        return links;
    }

    async _loadDocumentPreview(slug) {
        try {
            const response = await fetch(`/posts/${slug}.jsonld`);
            if (!response.ok) return null;

            const jsonld = await response.json();
            return {
                url: `/posts/${slug}`,
                title: jsonld.headline || jsonld.name || jsonld.title,
                description: jsonld.description
            };
        } catch (err) {
            console.error(`Failed to load preview for ${slug}:`, err);
            return null;
        }
    }

    _toggleMenu() {
        this._menuOpen = !this._menuOpen;
        if (this._menuOpen) {
            this._showMenu();
        } else {
            this._hideMenu();
        }
    }

    _showMenu() {
        // Create menu overlay
        const overlay = document.createElement('div');
        overlay.className = 'tc-menu-overlay';
        this._applyStyles(overlay, {
            position: 'fixed',
            top: '0',
            left: '0',
            width: '100%',
            height: '100%',
            background: 'rgba(0, 0, 0, 0.5)',
            zIndex: '999',
            display: 'flex',
            justifyContent: 'flex-start'
        });
        overlay.addEventListener('click', (e) => {
            if (e.target === overlay) {
                this._toggleMenu();
            }
        });

        // Create menu panel
        const menu = document.createElement('div');
        menu.className = 'tc-menu';
        this._applyStyles(menu, {
            background: '#fff',
            width: '300px',
            height: '100%',
            boxShadow: '2px 0 8px rgba(0,0,0,0.1)',
            display: 'flex',
            flexDirection: 'column',
            padding: '0'
        });

        // Menu header
        const menuHeader = document.createElement('div');
        this._applyStyles(menuHeader, {
            padding: '16px 24px',
            borderBottom: '1px solid #e1e4e8',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center'
        });

        const menuTitle = document.createElement('h3');
        menuTitle.textContent = 'Menu';
        this._applyStyles(menuTitle, {
            margin: '0',
            fontSize: '18px',
            fontWeight: 'bold'
        });
        menuHeader.appendChild(menuTitle);

        const closeBtn = document.createElement('button');
        closeBtn.textContent = '×';
        this._applyStyles(closeBtn, {
            background: 'transparent',
            border: 'none',
            fontSize: '28px',
            cursor: 'pointer',
            padding: '0',
            lineHeight: '1',
            color: '#586069'
        });
        closeBtn.addEventListener('click', () => this._toggleMenu());
        menuHeader.appendChild(closeBtn);

        menu.appendChild(menuHeader);

        // User status section (at top of menu items)
        const userSection = document.createElement('div');
        this._applyStyles(userSection, {
            padding: '16px 24px',
            borderBottom: '1px solid #e1e4e8',
            background: '#f6f8fa'
        });

        if (this._user) {
            // Show user info for logged-in users
            const userLabel = document.createElement('div');
            userLabel.textContent = 'Logged in as:';
            this._applyStyles(userLabel, {
                fontSize: '12px',
                color: '#586069',
                marginBottom: '8px'
            });
            userSection.appendChild(userLabel);

            const userInfo = document.createElement('div');
            this._applyStyles(userInfo, {
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
                marginBottom: '12px'
            });

            // GitHub icon
            const githubIcon = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
            githubIcon.setAttribute('height', '16');
            githubIcon.setAttribute('width', '16');
            githubIcon.setAttribute('viewBox', '0 0 16 16');
            githubIcon.setAttribute('fill', 'currentColor');
            const githubPath = document.createElementNS('http://www.w3.org/2000/svg', 'path');
            githubPath.setAttribute('d', 'M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z');
            githubIcon.appendChild(githubPath);
            userInfo.appendChild(githubIcon);

            const userName = document.createElement('span');
            userName.textContent = this._user.user_metadata?.user_name || this._user.email || 'User';
            this._applyStyles(userName, {
                fontSize: '14px',
                fontWeight: '500',
                color: '#24292e'
            });
            userInfo.appendChild(userName);
            userSection.appendChild(userInfo);

            // Logout button
            const logoutBtn = document.createElement('button');
            logoutBtn.textContent = 'Logout';
            this._applyStyles(logoutBtn, {
                width: '100%',
                padding: '8px 12px',
                fontSize: '14px',
                background: '#fafbfc',
                border: '1px solid #e1e4e8',
                borderRadius: '6px',
                cursor: 'pointer',
                transition: 'background 0.2s'
            });
            logoutBtn.addEventListener('mouseenter', () => {
                logoutBtn.style.background = '#f3f4f6';
            });
            logoutBtn.addEventListener('mouseleave', () => {
                logoutBtn.style.background = '#fafbfc';
            });
            logoutBtn.addEventListener('click', () => {
                this._toggleMenu();
                this._logout();
            });
            userSection.appendChild(logoutBtn);
        } else {
            // Show login button for non-logged-in users
            const loginLabel = document.createElement('div');
            loginLabel.textContent = 'Not logged in';
            this._applyStyles(loginLabel, {
                fontSize: '12px',
                color: '#586069',
                marginBottom: '8px'
            });
            userSection.appendChild(loginLabel);

            const loginBtn = document.createElement('button');
            loginBtn.textContent = 'Login with GitHub';
            this._applyStyles(loginBtn, {
                width: '100%',
                padding: '8px 12px',
                fontSize: '14px',
                fontWeight: '500',
                background: '#24292e',
                color: '#fff',
                border: 'none',
                borderRadius: '6px',
                cursor: 'pointer',
                transition: 'background 0.2s'
            });
            loginBtn.addEventListener('mouseenter', () => {
                loginBtn.style.background = '#444d56';
            });
            loginBtn.addEventListener('mouseleave', () => {
                loginBtn.style.background = '#24292e';
            });
            loginBtn.addEventListener('click', () => {
                this._toggleMenu();
                this._loginWithGitHub();
            });
            userSection.appendChild(loginBtn);
        }

        menu.appendChild(userSection);

        // Menu items
        const menuItems = document.createElement('div');
        this._applyStyles(menuItems, {
            padding: '8px 0'
        });

        const helpItem = document.createElement('button');
        helpItem.textContent = '❓ Help';
        this._applyStyles(helpItem, {
            width: '100%',
            padding: '12px 24px',
            background: 'transparent',
            border: 'none',
            textAlign: 'left',
            fontSize: '16px',
            cursor: 'pointer',
            transition: 'background 0.2s'
        });
        helpItem.addEventListener('mouseenter', () => {
            helpItem.style.background = '#f6f8fa';
        });
        helpItem.addEventListener('mouseleave', () => {
            helpItem.style.background = 'transparent';
        });
        helpItem.addEventListener('click', () => {
            this._toggleMenu();
            this._showHelp();
        });
        menuItems.appendChild(helpItem);

        // Latest Posts menu item
        const postsItem = document.createElement('button');
        postsItem.textContent = '📰 Latest Posts';
        this._applyStyles(postsItem, {
            width: '100%',
            padding: '12px 24px',
            background: 'transparent',
            border: 'none',
            textAlign: 'left',
            fontSize: '16px',
            cursor: 'pointer',
            transition: 'background 0.2s'
        });
        postsItem.addEventListener('mouseenter', () => {
            postsItem.style.background = '#f6f8fa';
        });
        postsItem.addEventListener('mouseleave', () => {
            postsItem.style.background = 'transparent';
        });
        postsItem.addEventListener('click', () => {
            this._toggleMenu();
            this._showLatestPosts();
        });
        menuItems.appendChild(postsItem);

        // GitHub repository link
        const githubItem = document.createElement('a');
        githubItem.href = 'https://github.com/stackdump/tens-city';
        githubItem.target = '_blank';
        githubItem.rel = 'noopener noreferrer';
        this._applyStyles(githubItem, {
            width: '100%',
            padding: '12px 24px',
            background: 'transparent',
            border: 'none',
            textAlign: 'left',
            fontSize: '16px',
            cursor: 'pointer',
            transition: 'background 0.2s',
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            textDecoration: 'none',
            color: 'inherit',
            boxSizing: 'border-box',
            overflow: 'hidden'
        });
        githubItem.addEventListener('mouseenter', () => {
            githubItem.style.background = '#f6f8fa';
        });
        githubItem.addEventListener('mouseleave', () => {
            githubItem.style.background = 'transparent';
        });
        
        // GitHub Octocat SVG logo
        const octocatSvg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
        octocatSvg.setAttribute('height', '16');
        octocatSvg.setAttribute('width', '16');
        octocatSvg.setAttribute('viewBox', '0 0 16 16');
        octocatSvg.setAttribute('fill', 'currentColor');
        // Prevent icon from expanding on hover
        octocatSvg.style.flexShrink = '0';
        octocatSvg.style.width = '16px';
        octocatSvg.style.height = '16px';
        const octocatPath = document.createElementNS('http://www.w3.org/2000/svg', 'path');
        octocatPath.setAttribute('d', 'M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z');
        octocatSvg.appendChild(octocatPath);
        githubItem.appendChild(octocatSvg);
        
        const githubText = document.createTextNode('GitHub Repository');
        githubItem.appendChild(githubText);
        menuItems.appendChild(githubItem);

        menu.appendChild(menuItems);
        overlay.appendChild(menu);
        this._root.appendChild(overlay);
    }

    _hideMenu() {
        const overlay = this._root.querySelector('.tc-menu-overlay');
        if (overlay) {
            overlay.remove();
        }
    }

    _showHelp() {
        // Create help overlay
        const overlay = document.createElement('div');
        overlay.className = 'tc-help-overlay';
        this._applyStyles(overlay, {
            position: 'fixed',
            top: '0',
            left: '0',
            width: '100%',
            height: '100%',
            background: 'rgba(0, 0, 0, 0.5)',
            zIndex: '1000',
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            padding: '24px'
        });
        overlay.addEventListener('click', (e) => {
            if (e.target === overlay) {
                this._hideHelp();
            }
        });

        // Create help panel
        const helpPanel = document.createElement('div');
        helpPanel.className = 'tc-help-panel';
        this._applyStyles(helpPanel, {
            background: '#fff',
            maxWidth: '700px',
            width: '100%',
            maxHeight: '90vh',
            borderRadius: '8px',
            boxShadow: '0 4px 16px rgba(0,0,0,0.2)',
            display: 'flex',
            flexDirection: 'column',
            overflow: 'hidden'
        });

        // Help header
        const helpHeader = document.createElement('div');
        this._applyStyles(helpHeader, {
            padding: '20px 24px',
            borderBottom: '1px solid #e1e4e8',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center'
        });

        const helpTitle = document.createElement('h2');
        helpTitle.textContent = 'Help';
        this._applyStyles(helpTitle, {
            margin: '0',
            fontSize: '24px',
            fontWeight: 'bold'
        });
        helpHeader.appendChild(helpTitle);

        const closeBtn = document.createElement('button');
        closeBtn.textContent = '×';
        this._applyStyles(closeBtn, {
            background: 'transparent',
            border: 'none',
            fontSize: '32px',
            cursor: 'pointer',
            padding: '0',
            lineHeight: '1',
            color: '#586069'
        });
        closeBtn.addEventListener('click', () => this._hideHelp());
        helpHeader.appendChild(closeBtn);

        helpPanel.appendChild(helpHeader);

        // Help content
        const helpContent = document.createElement('div');
        this._applyStyles(helpContent, {
            padding: '24px',
            overflowY: 'auto',
            flex: '1'
        });

        // Build help content
        const sections = [
            {
                title: 'What is Tens City?',
                content: 'Tens City is a content-addressable storage system for JSON-LD documents. Create, save, and share structured data with cryptographic integrity using Content Identifiers (CIDs).',
                link: {
                    text: 'Learn more at GitHub',
                    url: 'https://github.com/stackdump/tens-city'
                }
            },
            {
                title: 'What is JSON-LD?',
                content: 'JSON-LD (JavaScript Object Notation for Linked Data) is a format for expressing structured data using JSON with semantic meaning. It uses vocabularies like schema.org to create machine-readable data that can be shared across the web.',
                link: {
                    text: 'Learn more at schema.org',
                    url: 'https://schema.org'
                }
            },
            {
                title: 'Authentication',
                content: 'Use GitHub OAuth to authenticate and save your work. After logging in, your JSON-LD documents will be associated with your GitHub identity, allowing you to delete your own saved objects.'
            },
            {
                title: 'Saving Your Work',
                content: 'Click the Save button to:\n\n• Validate your JSON-LD (requires @context field)\n• Generate a content-addressed identifier (CID)\n• Store the object permanently\n• Update the URL to ?cid=...\n\nYou must be logged in to save. Anyone can view saved objects via ?cid= URLs.'
            },
            {
                title: 'Using the Editor',
                content: 'The editor provides:\n\n• Syntax highlighting for JSON\n• Clear button to reset content\n• Permalink button to share current data\n• Auto-updating <script type="application/ld+json"> tag\n• Delete button (visible only for objects you own)'
            }
        ];

        sections.forEach(section => {
            const sectionTitle = document.createElement('h3');
            sectionTitle.textContent = section.title;
            this._applyStyles(sectionTitle, {
                fontSize: '18px',
                fontWeight: 'bold',
                marginTop: '16px',
                marginBottom: '8px',
                color: '#24292e'
            });
            helpContent.appendChild(sectionTitle);

            const sectionContent = document.createElement('p');
            sectionContent.textContent = section.content;
            this._applyStyles(sectionContent, {
                fontSize: '14px',
                lineHeight: '1.6',
                color: '#586069',
                marginBottom: '16px',
                whiteSpace: 'pre-wrap'
            });
            helpContent.appendChild(sectionContent);

            // Add link if present
            if (section.link) {
                const linkElement = document.createElement('a');
                linkElement.href = section.link.url;
                linkElement.textContent = section.link.text;
                linkElement.target = '_blank';
                linkElement.rel = 'noopener noreferrer';
                this._applyStyles(linkElement, {
                    fontSize: '14px',
                    color: '#0366d6',
                    textDecoration: 'none',
                    display: 'block',
                    marginBottom: '16px'
                });
                linkElement.addEventListener('mouseenter', () => {
                    linkElement.style.textDecoration = 'underline';
                });
                linkElement.addEventListener('mouseleave', () => {
                    linkElement.style.textDecoration = 'none';
                });
                helpContent.appendChild(linkElement);
            }
        });

        helpPanel.appendChild(helpContent);
        overlay.appendChild(helpPanel);
        this._root.appendChild(overlay);
    }

    _hideHelp() {
        const overlay = this._root.querySelector('.tc-help-overlay');
        if (overlay) {
            overlay.remove();
        }
    }

    async _showLatestPosts() {
        // Create overlay for modal
        const overlay = document.createElement('div');
        overlay.className = 'tc-posts-overlay';
        this._applyStyles(overlay, {
            position: 'fixed',
            top: '0',
            left: '0',
            width: '100%',
            height: '100%',
            background: 'rgba(0, 0, 0, 0.5)',
            zIndex: '1000',
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            padding: '24px'
        });
        overlay.addEventListener('click', (e) => {
            if (e.target === overlay) {
                this._hideLatestPosts();
            }
        });

        // Create modal panel
        const postsPanel = document.createElement('div');
        postsPanel.className = 'tc-posts-panel';
        this._applyStyles(postsPanel, {
            background: '#fff',
            maxWidth: '800px',
            width: '100%',
            maxHeight: '90vh',
            borderRadius: '8px',
            boxShadow: '0 4px 16px rgba(0,0,0,0.2)',
            display: 'flex',
            flexDirection: 'column',
            overflow: 'hidden'
        });

        // Header
        const postsHeader = document.createElement('div');
        this._applyStyles(postsHeader, {
            padding: '20px 24px',
            borderBottom: '1px solid #e1e4e8',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center'
        });

        const postsTitle = document.createElement('h2');
        postsTitle.textContent = 'Latest Posts';
        this._applyStyles(postsTitle, {
            margin: '0',
            fontSize: '24px',
            fontWeight: 'bold'
        });
        postsHeader.appendChild(postsTitle);

        const closeBtn = document.createElement('button');
        closeBtn.textContent = '×';
        this._applyStyles(closeBtn, {
            background: 'transparent',
            border: 'none',
            fontSize: '32px',
            cursor: 'pointer',
            padding: '0',
            lineHeight: '1',
            color: '#586069'
        });
        closeBtn.addEventListener('click', () => this._hideLatestPosts());
        postsHeader.appendChild(closeBtn);

        postsPanel.appendChild(postsHeader);

        // Content area
        const postsContent = document.createElement('div');
        postsContent.className = 'tc-posts-content';
        this._applyStyles(postsContent, {
            flex: '1',
            overflowY: 'auto',
            padding: '24px'
        });

        // Show loading state
        postsContent.innerHTML = '<p style="color: #586069;">Loading latest posts...</p>';

        // Load and render posts
        try {
            const response = await fetch('/posts/index.jsonld');
            if (response.ok) {
                const index = await response.json();
                this._renderLatestPosts(postsContent, index);
            } else {
                postsContent.innerHTML = '<p style="color: #999;">No posts available</p>';
            }
        } catch (err) {
            console.error('Failed to load posts index:', err);
            postsContent.innerHTML = '<p style="color: #d73a49;">Failed to load posts</p>';
        }

        postsPanel.appendChild(postsContent);
        overlay.appendChild(postsPanel);
        this._root.appendChild(overlay);
    }

    _renderLatestPosts(container, index) {
        // Clear container
        container.innerHTML = '';

        // Get posts from index
        const items = index.itemListElement || [];
        
        if (items.length === 0) {
            container.innerHTML = '<p style="color: #999;">No posts found</p>';
            return;
        }

        // Create card-based layout for posts
        items.forEach((item, idx) => {
            const post = item.item || item;
            const postCard = document.createElement('div');
            this._applyStyles(postCard, {
                marginBottom: idx < items.length - 1 ? '16px' : '0',
                padding: '20px',
                background: '#f6f8fa',
                borderRadius: '6px',
                border: '1px solid #e1e4e8',
                cursor: 'pointer',
                transition: 'all 0.2s'
            });

            postCard.addEventListener('mouseenter', () => {
                postCard.style.background = '#e1e4e8';
                postCard.style.transform = 'translateY(-2px)';
                postCard.style.boxShadow = '0 2px 8px rgba(0,0,0,0.1)';
            });
            postCard.addEventListener('mouseleave', () => {
                postCard.style.background = '#f6f8fa';
                postCard.style.transform = 'translateY(0)';
                postCard.style.boxShadow = 'none';
            });

            const title = document.createElement('h3');
            title.textContent = post.headline || post.name || 'Untitled';
            this._applyStyles(title, {
                margin: '0 0 8px 0',
                fontSize: '18px',
                fontWeight: '600',
                color: '#24292e'
            });
            postCard.appendChild(title);

            if (post.description) {
                const desc = document.createElement('p');
                desc.textContent = post.description;
                this._applyStyles(desc, {
                    margin: '0 0 12px 0',
                    fontSize: '14px',
                    color: '#586069',
                    lineHeight: '1.5'
                });
                postCard.appendChild(desc);
            }

            // Add date if available
            if (post.datePublished) {
                const date = document.createElement('div');
                const dateObj = new Date(post.datePublished);
                date.textContent = `📅 ${dateObj.toLocaleDateString()}`;
                this._applyStyles(date, {
                    fontSize: '12px',
                    color: '#6a737d',
                    marginBottom: '8px'
                });
                postCard.appendChild(date);
            }

            // Add "Read more" link
            const readMore = document.createElement('a');
            const url = item.url || post.url || post['@id'];
            if (url) {
                // Extract slug from URL
                let slug = url;
                if (url.includes('/posts/')) {
                    slug = url.split('/posts/').pop().split(/[?#]/)[0];
                }
                readMore.href = `/posts/${slug}`;
                readMore.target = '_blank';
                readMore.textContent = 'Read more →';
                this._applyStyles(readMore, {
                    fontSize: '14px',
                    color: '#0366d6',
                    textDecoration: 'none',
                    fontWeight: '500'
                });
                readMore.addEventListener('mouseenter', () => {
                    readMore.style.textDecoration = 'underline';
                });
                readMore.addEventListener('mouseleave', () => {
                    readMore.style.textDecoration = 'none';
                });
                postCard.appendChild(readMore);

                // Make the whole card clickable
                postCard.addEventListener('click', (e) => {
                    if (e.target !== readMore) {
                        window.open(`/posts/${slug}`, '_blank');
                    }
                });
            }

            container.appendChild(postCard);
        });
    }

    _hideLatestPosts() {
        const overlay = this._root.querySelector('.tc-posts-overlay');
        if (overlay) {
            overlay.remove();
        }
    }

    async _showLatestPostView() {
        // Redirect to the latest post instead of showing it inline
        try {
            const response = await fetch('/posts/index.jsonld');
            if (!response.ok) {
                // Failed to fetch index - stay on current page
                return;
            }
            
            const index = await response.json();
            const items = index.itemListElement || [];
            
            if (items.length === 0) {
                // No posts available - stay on current page
                return;
            }
            
            const latestItem = items[0];
            const post = latestItem.item || latestItem;
            const urlString = post.url || post['@id'];
            
            if (!urlString) {
                // No URL found - stay on current page
                return;
            }
            
            // Extract slug from URL using proper URL parsing
            try {
                // Handle both absolute and relative URLs
                const url = urlString.startsWith('http') ? new URL(urlString) : new URL(urlString, window.location.origin);
                const pathParts = url.pathname.split('/posts/');
                
                if (pathParts.length > 1) {
                    const slug = pathParts[pathParts.length - 1].replace(/\/$/, ''); // Remove trailing slash
                    if (slug) {
                        // Redirect to the latest post
                        window.location.href = `/posts/${slug}`;
                    }
                }
            } catch (err) {
                console.error('Failed to parse post URL:', err);
                // Failed to parse URL - stay on current page
            }
        } catch (err) {
            console.error('Failed to load latest post for redirect:', err);
            // Don't redirect on error - just continue showing the editor
        }
    }

    _clearEditor() {
        if (this._editorMode === 'jsonld' && this._aceEditor) {
            this._aceEditor.session.setValue('{\n  "@context": "https://pflow.xyz/schema",\n  "@type": "Object"\n}');
        } else if (this._editorMode === 'markdown' && this._markdownEditor) {
            this._markdownEditor.value = '';
            this._docTags = [];
            const form = this._frontmatterForm;
            if (form) {
                form.querySelector('#fm-title').value = '';
                form.querySelector('#fm-description').value = '';
                form.querySelector('#fm-slug').value = '';
                const now = new Date().toISOString().slice(0, 16);
                form.querySelector('#fm-datePublished').value = now;
                form.querySelector('#fm-dateModified').value = '';
            }
            this._updateMarkdownPreview();
            // Clear saved CID/slug when clearing markdown editor
            this._lastSavedCid = null;
            this._lastSavedSlug = null;
            this._updatePermalinkAnchor();
        }
    }

    _toggleEditorMode() {
        // Switch between jsonld and markdown modes
        this._editorMode = this._editorMode === 'jsonld' ? 'markdown' : 'jsonld';
        
        // Clear saved CID/slug when switching modes
        this._lastSavedCid = null;
        this._lastSavedSlug = null;
        
        // Update toggle button text
        const toggleBtn = this._root.querySelector('#tc-mode-toggle');
        if (toggleBtn) {
            toggleBtn.textContent = this._editorMode === 'jsonld' ? '📝 Switch to Markdown' : '📋 Switch to JSON-LD';
        }

        // Remove existing editor container
        const editorContainer = this._appContainer.querySelector('.tc-editor-container');
        if (editorContainer) {
            editorContainer.remove();
        }

        // Recreate appropriate editor and show it (user is already using the app)
        this._createEditor().then(() => {
            this._showEditorContainer();
        }).catch(err => {
            console.error('Failed to recreate editor:', err);
            // Editor failed to load, but at least show a message in the container
            const errorDiv = document.createElement('div');
            errorDiv.textContent = 'Failed to load editor. Please refresh the page.';
            errorDiv.style.padding = '24px';
            errorDiv.style.color = '#d73a49';
            this._appContainer.appendChild(errorDiv);
        });
    }

    _loadFromScriptTag() {
        // Look for script tag with type="application/ld+json" inside tens-city element
        const scriptTag = this.querySelector('script[type="application/ld+json"]');
        if (scriptTag && scriptTag.textContent) {
            try {
                // Parse and re-stringify to ensure it's valid JSON
                const data = JSON.parse(scriptTag.textContent);
                return JSON.stringify(data, null, 2);
            } catch (err) {
                console.error('Failed to parse script tag JSON:', err);
                return null;
            }
        }
        return null;
    }

    _updateScriptTag() {
        if (!this._aceEditor) return;

        try {
            const editorContent = this._aceEditor.session.getValue();
            // Validate it's valid JSON before updating
            JSON.parse(editorContent);

            // Remove existing script tag if present
            const existingTag = this.querySelector('script[type="application/ld+json"]');
            if (existingTag) {
                existingTag.remove();
            }

            // Create new script tag with current editor content
            const scriptTag = document.createElement('script');
            scriptTag.type = 'application/ld+json';
            scriptTag.textContent = editorContent;
            this.appendChild(scriptTag);
        } catch (err) {
            // Silently ignore invalid JSON - don't update script tag
        }
    }

    async _saveData() {
        console.log('Save: User clicked save button');
        
        if (!this._user) {
            alert('Please log in to save data');
            return;
        }

        if (this._editorMode === 'jsonld') {
            await this._saveJSONLD();
        } else {
            await this._saveMarkdown();
        }
    }

    async _saveJSONLD() {
        if (!this._aceEditor) {
            console.error('Save: Editor not initialized');
            return;
        }

        try {
            const editorContent = this._aceEditor.session.getValue();
            const data = JSON.parse(editorContent);

            // Validate it's valid JSON-LD (has @context)
            if (!data['@context']) {
                alert('Invalid JSON-LD: missing @context field');
                return;
            }

            console.log('Save: Valid JSON-LD, sending to /api/save');
            
            // Use canonical JSON encoding to ensure consistent CID calculation
            const canonicalData = canonicalJSON(data);
            
            // Get the session token for authentication
            const { data: { session } } = await this._supabase.auth.getSession();
            const authToken = session?.access_token;
            
            if (!authToken) {
                console.error('Save: No auth token available');
                alert('Authentication token not available. Please log in again.');
                return;
            }
            
            console.log('Save: Sending save request to /api/save');
            
            // Use API endpoint to save
            const response = await fetch('/api/save', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${authToken}`,
                },
                body: canonicalData
            });

            if (!response.ok) {
                const errorText = await response.text();
                console.error('Save: Failed with status', response.status, errorText);
                alert(`Save failed: ${response.statusText}`);
                return;
            }

            const result = await response.json();
            const cid = result.cid;
            
            console.log('Save: Success! CID:', cid, '- Updating URL');
            
            // Update URL without reloading page
            const url = new URL(window.location.origin + window.location.pathname);
            url.searchParams.set('cid', cid);
            window.history.pushState({}, '', url.toString());
            
            // Cache ownership immediately after save since we know the current user owns it
            this._ownershipCache[cid] = true;
            
            // Show delete button since we now have a CID and we own it
            const deleteBtn = this._root?.querySelector('#tc-delete-btn');
            if (deleteBtn) {
                deleteBtn.style.display = 'inline-block';
            }
            
            // Show success message
            alert(`Saved successfully! CID: ${cid}`);
            
        } catch (err) {
            console.error('Save: Exception occurred:', err);
            alert(`Save failed: ${err.message}`);
        }
    }

    async _saveMarkdown() {
        if (!this._markdownEditor || !this._frontmatterForm) {
            console.error('Save: Markdown editor not initialized');
            return;
        }

        try {
            // Get form data
            const form = this._frontmatterForm;
            const title = form.querySelector('#fm-title').value;
            const description = form.querySelector('#fm-description').value;
            const slug = form.querySelector('#fm-slug').value;
            const datePublished = form.querySelector('#fm-datePublished').value;
            const dateModified = form.querySelector('#fm-dateModified').value;

            // Validate required fields (author is no longer required from client)
            if (!title || !datePublished || !slug) {
                alert('Please fill in all required fields (Title, Date Published, Slug)');
                return;
            }

            // Build frontmatter (author will be set server-side from authenticated user)
            const frontmatter = {
                title: title,
                datePublished: new Date(datePublished).toISOString(),
                lang: 'en'
            };

            if (description) frontmatter.description = description;
            if (dateModified) frontmatter.dateModified = new Date(dateModified).toISOString();

            const markdown = this._markdownEditor.value;

            // Get the session token for authentication
            const { data: { session } } = await this._supabase.auth.getSession();
            const authToken = session?.access_token;
            
            if (!authToken) {
                console.error('Save: No auth token available');
                alert('Authentication token not available. Please log in again.');
                return;
            }

            console.log('Save: Sending markdown save request to /api/posts/save');

            // Send to /api/posts/save endpoint
            const response = await fetch('/api/posts/save', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${authToken}`,
                },
                body: JSON.stringify({
                    slug: slug,
                    content: markdown,
                    frontmatter: frontmatter
                })
            });

            if (!response.ok) {
                const errorText = await response.text();
                console.error('Save: Failed with status', response.status, errorText);
                alert(`Save failed: ${response.statusText}`);
                return;
            }

            const result = await response.json();
            const cid = result.cid;

            console.log('Save: Success! CID:', cid, 'Slug:', slug);

            // Store for permalink
            this._lastSavedCid = cid;
            this._lastSavedSlug = slug;
            this._updatePermalinkAnchor();

            // Show success message
            alert(`Document saved successfully!\nCID: ${cid}\nSlug: ${slug}\nView at: /posts/${slug}`);

        } catch (err) {
            console.error('Save: Exception occurred:', err);
            alert(`Save failed: ${err.message}`);
        }
    }

    async _deleteObject() {
        console.log('Delete: User clicked delete button');
        
        if (!this._user) {
            alert('Please log in to delete data');
            return;
        }

        // Get the current CID from URL
        const urlParams = new URLSearchParams(window.location.search);
        const cidParam = urlParams.get('cid');
        
        if (!cidParam) {
            alert('No object to delete. Save an object first to get a CID.');
            return;
        }

        // Confirm deletion
        if (!confirm(`Are you sure you want to delete this object?\n\nCID: ${cidParam}\n\nThis action cannot be undone.`)) {
            return;
        }

        try {
            console.log('Delete: Sending delete request for CID:', cidParam);
            
            // Get the session token for authentication
            const { data: { session } } = await this._supabase.auth.getSession();
            const authToken = session?.access_token;
            
            if (!authToken) {
                console.error('Delete: No auth token available');
                alert('Authentication token not available. Please log in again.');
                return;
            }
            
            console.log('Delete: Sending delete request to /o/' + cidParam);
            
            // Send delete request
            const response = await fetch('/o/' + cidParam, {
                method: 'DELETE',
                headers: {
                    'Authorization': `Bearer ${authToken}`,
                }
            });

            if (!response.ok) {
                const errorText = await response.text();
                console.error('Delete: Failed with status', response.status, errorText);
                
                if (response.status === 403) {
                    alert('Delete failed: Only the author can delete this object.');
                } else if (response.status === 404) {
                    alert('Delete failed: Object not found.');
                } else {
                    alert(`Delete failed: ${response.statusText}`);
                }
                return;
            }
            
            console.log('Delete: Success! Object deleted');
            
            // Clear the editor and URL
            this._clearEditor();
            const url = new URL(window.location.origin + window.location.pathname);
            window.history.pushState({}, '', url.toString());
            
            // Hide delete button
            await this._updateDeleteButtonVisibility();
            
            // Show success message
            alert('Object deleted successfully!');
            
        } catch (err) {
            console.error('Delete: Exception occurred:', err);
            alert(`Delete failed: ${err.message}`);
        }
    }

    async _updateDeleteButtonVisibility(loadedData = null) {
        // Show delete button only when viewing a CID, user is authenticated, and user owns the object
        const deleteBtn = this._root?.querySelector('#tc-delete-btn');
        if (!deleteBtn) {
            return;
        }

        const urlParams = new URLSearchParams(window.location.search);
        const cidParam = urlParams.get('cid');
        
        if (!cidParam || !this._user) {
            deleteBtn.style.display = 'none';
            return;
        }

        // Check cache first to avoid redundant checks
        if (this._ownershipCache.hasOwnProperty(cidParam)) {
            deleteBtn.style.display = this._ownershipCache[cidParam] ? 'inline-block' : 'none';
            return;
        }

        // Check ownership from loaded data (if provided) or from editor content
        try {
            let data = loadedData;
            
            // If data not provided, try to parse from editor
            if (!data && this._aceEditor) {
                try {
                    const editorContent = this._aceEditor.session.getValue();
                    data = JSON.parse(editorContent);
                } catch (err) {
                    // Could not parse editor content
                }
            }
            
            if (!data) {
                this._ownershipCache[cidParam] = false;
                deleteBtn.style.display = 'none';
                return;
            }
            
            // Extract author information from the data
            const author = data.author;
            if (!author) {
                this._ownershipCache[cidParam] = false;
                deleteBtn.style.display = 'none';
                return;
            }
            
            // Get current user's GitHub info from this._user (already set during auth)
            if (!this._user) {
                this._ownershipCache[cidParam] = false;
                deleteBtn.style.display = 'none';
                return;
            }
            
            const userMetadata = this._user.user_metadata || {};
            const currentUserName = userMetadata.user_name || userMetadata.preferred_username;
            const currentGitHubID = userMetadata.provider_id || userMetadata.sub;
            
            // Check if current user is the author
            // Compare GitHub ID (strip "github:" prefix if present)
            const authorGitHubID = author.id ? author.id.replace(/^github:/, '') : null;
            const authorUserName = author.name;
            
            const isOwned = (currentGitHubID && authorGitHubID && currentGitHubID === authorGitHubID) ||
                           (currentUserName && authorUserName && currentUserName === authorUserName);
            
            this._ownershipCache[cidParam] = isOwned;
            deleteBtn.style.display = isOwned ? 'inline-block' : 'none';
        } catch (err) {
            console.error('Failed to check ownership:', err);
            this._ownershipCache[cidParam] = false;
            deleteBtn.style.display = 'none';
        }
    }

    _loadFromURL() {
        // Check for data in URL parameter
        const urlParams = new URLSearchParams(window.location.search);
        const encodedData = urlParams.get('data');
        
        if (encodedData) {
            const permalinkData = this._decodePermalinkData(encodedData);
            if (permalinkData) {
                // Update script tag with loaded data
                const scriptTag = document.createElement('script');
                scriptTag.type = 'application/ld+json';
                scriptTag.textContent = permalinkData.jsonString;
                this.appendChild(scriptTag);
                return permalinkData;
            }
        }
        return null;
    }

    _updatePermalinkAnchor() {
        if (!this._permalinkAnchor) return;

        if (this._editorMode === 'jsonld') {
            // JSON-LD mode: create permalink with data in URL
            if (!this._aceEditor) return;
            
            try {
                const editorContent = this._aceEditor.session.getValue();
                // Parse and canonicalize JSON to ensure consistent encoding
                const parsed = JSON.parse(editorContent);
                const canonical = canonicalJSON(parsed);

                // Create URL with data parameter - use clean URL without existing query params
                const url = new URL(window.location.origin + window.location.pathname);
                url.searchParams.set('data', encodeURIComponent(canonical));
                
                // Update anchor href and clear any custom title
                this._permalinkAnchor.href = url.toString();
                this._permalinkAnchor.title = 'Link to current data';
                console.log('Permalink: Updated permalink anchor with current editor content');
            } catch (err) {
                // If JSON is invalid, set href to # to prevent navigation
                this._permalinkAnchor.href = '#';
                this._permalinkAnchor.title = 'Fix JSON errors to enable permalink';
                console.log('Permalink: Invalid JSON, permalink disabled');
            }
        } else {
            // Markdown mode: link to saved CID if available
            if (this._lastSavedCid) {
                // Link to the immutable object by CID
                this._permalinkAnchor.href = `/o/${this._lastSavedCid}`;
                this._permalinkAnchor.title = 'Link to saved object';
                console.log('Permalink: Updated to saved object CID:', this._lastSavedCid);
            } else {
                // No saved content yet - disable permalink
                this._permalinkAnchor.href = '#';
                this._permalinkAnchor.title = 'Save the post first to get a permalink';
                console.log('Permalink: Disabled (no saved content)');
            }
        }
    }

    async _loadInitialData() {
        console.log('Loading initial data...');
        
        // Check for CID parameter first
        const urlParams = new URLSearchParams(window.location.search);
        const cidParam = urlParams.get('cid');
        
        if (cidParam && this._aceEditor) {
            console.log('Loading data from CID:', cidParam);
            // Load object by CID from API
            try {
                const response = await fetch(`/o/${cidParam}`);
                if (response.ok) {
                    const data = await response.json();
                    this._aceEditor.session.setValue(JSON.stringify(data, null, 2));
                    this._updatePermalinkAnchor();
                    // Pass the loaded data to ownership check to avoid API call
                    await this._updateDeleteButtonVisibility(data);
                    console.log('Successfully loaded data from CID');
                    return;
                } else {
                    console.error('Failed to load CID:', response.status, response.statusText);
                }
            } catch (err) {
                console.error('Failed to load object by CID:', err);
            }
        }

        // Check for pending permalink data captured before authentication
        if (this._pendingPermalinkData) {
            console.log('Loading data from pending permalink data');
            const urlData = this._pendingPermalinkData;
            
            // Load the data into editor
            if (urlData.jsonString && this._aceEditor) {
                this._aceEditor.session.setValue(urlData.jsonString);
                this._updatePermalinkAnchor();
                await this._updateDeleteButtonVisibility();
                console.log('Successfully loaded permalink data into editor');
            }
            
            // Clear the pending data now that we're using it
            this._pendingPermalinkData = null;
            sessionStorage.removeItem('pendingPermalinkData');
            
            return;
        }

        // Fallback: Check for permalink data in URL (edge case where early capture failed)
        const urlData = this._loadFromURL();
        if (urlData) {
            console.log('Loading data from URL parameter (fallback path)');
            
            // Load the data into editor
            if (urlData.jsonString && this._aceEditor) {
                this._aceEditor.session.setValue(urlData.jsonString);
                this._updatePermalinkAnchor();
                console.log('Successfully loaded URL data into editor');
            }
            
            return;
        }

        // Check for script tag data
        const scriptData = this._loadFromScriptTag();
        if (scriptData && this._aceEditor) {
            console.log('Loading data from script tag');
            this._aceEditor.session.setValue(scriptData);
            this._updatePermalinkAnchor();
            console.log('Successfully loaded script tag data');
            return;
        }

        // Use default template
        console.log('Using default template');
        if (this._aceEditor) {
            this._updatePermalinkAnchor();
        }
    }

    _applyStyles(el, styles = {}) {
        Object.assign(el.style, styles);
    }
}

customElements.define('tens-city', TensCity);
export { TensCity };
