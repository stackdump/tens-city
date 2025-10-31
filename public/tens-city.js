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
                const permalinkData = {
                    jsonString: JSON.stringify(data, null, 2),
                    data: data
                };
                
                // Store in both instance variable and sessionStorage
                // sessionStorage persists across OAuth redirects
                this._pendingPermalinkData = permalinkData;
                sessionStorage.setItem('pendingPermalinkData', JSON.stringify(permalinkData));
                console.log('Permalink: Successfully parsed and stored permalink data (in memory and sessionStorage)');
            } catch (err) {
                console.error('Permalink: Failed to parse URL data:', err);
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
        link.href = 'README.md';
        link.target = '_blank';
        link.textContent = 'README.md';
        helpText.appendChild(link);
        helpText.appendChild(document.createTextNode(' for setup instructions.'));
        errorContainer.appendChild(helpText);

        this._appContainer.appendChild(errorContainer);
    }

    _showLogin() {
        this._loginContainer.style.display = 'flex';
        this._appContainer.style.display = 'none';
        this._loginContainer.innerHTML = '';

        const title = document.createElement('h1');
        title.textContent = 'Tens City';
        this._applyStyles(title, {
            fontSize: '48px',
            fontWeight: 'bold',
            margin: '0',
            color: '#333'
        });
        this._loginContainer.appendChild(title);

        const subtitle = document.createElement('p');
        subtitle.textContent = 'A place for your JSON-LD objects';
        this._applyStyles(subtitle, {
            fontSize: '18px',
            color: '#666',
            margin: '10px 0 40px 0'
        });
        this._loginContainer.appendChild(subtitle);

        const loginBtn = document.createElement('button');
        loginBtn.textContent = 'Login with GitHub';
        loginBtn.className = 'tc-login-btn';
        this._applyStyles(loginBtn, {
            padding: '12px 24px',
            fontSize: '16px',
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
        loginBtn.addEventListener('click', () => this._loginWithGitHub());
        this._loginContainer.appendChild(loginBtn);
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
        this._loginContainer.style.display = 'none';
        this._appContainer.style.display = 'flex';
        this._appContainer.innerHTML = '';
        
        this._createHeader();
        this._createToolbar();
        await this._createEditor();
        await this._loadInitialData();
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

        const userInfo = document.createElement('div');
        this._applyStyles(userInfo, {
            display: 'flex',
            alignItems: 'center',
            gap: '16px'
        });

        const userEmail = document.createElement('span');
        userEmail.textContent = this._user?.email || this._user?.user_metadata?.user_name || 'User';
        this._applyStyles(userEmail, {
            fontSize: '14px',
            color: '#586069'
        });
        userInfo.appendChild(userEmail);

        const logoutBtn = document.createElement('button');
        logoutBtn.textContent = 'Logout';
        this._applyStyles(logoutBtn, {
            padding: '6px 12px',
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
        logoutBtn.addEventListener('click', () => this._logout());
        userInfo.appendChild(logoutBtn);

        header.appendChild(userInfo);
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

        const clearBtn = makeButton('🗑️ Clear', 'Clear editor', () => this._clearEditor());

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
        const editorContainer = document.createElement('div');
        editorContainer.className = 'tc-editor-container';
        this._applyStyles(editorContainer, {
            flex: '1 1 auto',
            display: 'flex',
            flexDirection: 'column',
            padding: '24px',
            overflow: 'hidden'
        });

        const editorWrapper = document.createElement('div');
        editorWrapper.className = 'tc-editor-wrapper';
        this._applyStyles(editorWrapper, {
            width: '100%',
            height: '100%',
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
                title: 'GitHub Authentication',
                content: 'Tens City uses GitHub OAuth for authentication. When you click "Login with GitHub", you\'ll be redirected to GitHub to authorize the application. After authorization, you\'ll be redirected back to Tens City with your GitHub identity. This allows the application to associate your data with your GitHub username.'
            },
            {
                title: 'User Namespaces and Slugs',
                content: 'Each user has their own namespace based on their GitHub username. Within that namespace, you can create "slugs" - unique identifiers for your JSON-LD objects. For example, if your GitHub username is "alice" and you create a slug called "my-project", your objects will be accessible at /u/alice/g/my-project/latest. This provides a human-readable way to organize and access your data.'
            },
            {
                title: 'Auto-Save with data=XXX',
                content: 'When you visit a URL with the data parameter (e.g., ?data=...), Tens City automatically saves the JSON-LD object if:\n\n• You are logged in\n• The JSON is valid JSON-LD (has an @context field)\n\nAfter saving, the app redirects you to a ?cid=... URL that references the permanent, content-addressed version of your data. This makes it easy to share and preserve JSON-LD objects just by sharing a link.'
            },
            {
                title: 'Using the Editor',
                content: 'The editor allows you to create and edit JSON-LD documents. Use the Clear button to reset the editor. The Permalink button creates a shareable link with your current editor content. The editor automatically updates the embedded <script type="application/ld+json"> tag in the page as you type.'
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

    _clearEditor() {
        if (!this._aceEditor) return;
        this._aceEditor.session.setValue('{\n  "@context": "https://pflow.xyz/schema",\n  "@type": "Object"\n}');
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

    _loadFromURL() {
        // Check for data in URL parameter
        const urlParams = new URLSearchParams(window.location.search);
        const encodedData = urlParams.get('data');
        
        if (encodedData) {
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
                // Update script tag with loaded data
                const scriptTag = document.createElement('script');
                scriptTag.type = 'application/ld+json';
                scriptTag.textContent = JSON.stringify(data, null, 2);
                this.appendChild(scriptTag);
                return {
                    jsonString: JSON.stringify(data, null, 2),
                    data: data
                };
            } catch (err) {
                console.error('Failed to parse URL data:', err);
                return null;
            }
        }
        return null;
    }

    async _autoSaveFromURL(data) {
        console.log('Auto-save: Checking if auto-save should run...');
        
        // Auto-save feature: if user is authenticated and data has valid JSON-LD, save it
        if (!this._user || !data || !data.data) {
            console.log('Auto-save: Skipping - user not authenticated or no data');
            return null;
        }

        // Validate it's valid JSON-LD (has @context)
        if (!data.data['@context']) {
            console.log('Auto-save: Skipping - missing @context');
            return null;
        }

        console.log('Auto-save: Attempting to auto-save JSON-LD data');
        
        try {
            // Use canonical JSON encoding to ensure consistent CID calculation
            const canonicalData = canonicalJSON(data.data);
            
            // Get the session token for authentication
            const { data: { session } } = await this._supabase.auth.getSession();
            const authToken = session?.access_token;
            
            if (!authToken) {
                console.error('Auto-save: No auth token available');
                return null;
            }
            
            console.log('Auto-save: Sending save request to /api/save');
            
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
                console.error('Auto-save: Failed with status', response.status, response.statusText);
                return null;
            }

            const result = await response.json();
            const cid = result.cid;
            
            console.log('Auto-save: Success! CID:', cid, '- Redirecting to ?cid= URL');
            
            // Redirect to ?cid= URL
            const url = new URL(window.location.origin + window.location.pathname);
            url.searchParams.set('cid', cid);
            window.location.href = url.toString();
            
            return cid;
        } catch (err) {
            console.error('Auto-save: Exception occurred:', err);
            return null;
        }
    }

    _updatePermalinkAnchor() {
        if (!this._aceEditor || !this._permalinkAnchor) return;

        try {
            const editorContent = this._aceEditor.session.getValue();
            // Parse and canonicalize JSON to ensure consistent encoding
            const parsed = JSON.parse(editorContent);
            const canonical = canonicalJSON(parsed);

            // Create URL with data parameter - use clean URL without existing query params
            const url = new URL(window.location.origin + window.location.pathname);
            url.searchParams.set('data', encodeURIComponent(canonical));
            
            // Update anchor href
            this._permalinkAnchor.href = url.toString();
            console.log('Permalink: Updated permalink anchor with current editor content');
        } catch (err) {
            // If JSON is invalid, set href to # to prevent navigation
            this._permalinkAnchor.href = '#';
            console.log('Permalink: Invalid JSON, permalink disabled');
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
            
            // Clear the pending data now that we're using it
            this._pendingPermalinkData = null;
            sessionStorage.removeItem('pendingPermalinkData');
            
            // Auto-save if user is authenticated
            await this._autoSaveFromURL(urlData);
            
            // If auto-save redirected, we won't reach here
            // Otherwise, just load the data into editor
            if (urlData.jsonString && this._aceEditor) {
                this._aceEditor.session.setValue(urlData.jsonString);
                this._updatePermalinkAnchor();
                console.log('Successfully loaded permalink data into editor');
                return;
            }
        }

        // Check for permalink data in URL (fallback if not already captured)
        const urlData = this._loadFromURL();
        if (urlData) {
            console.log('Loading data from URL parameter');
            // Auto-save if user is authenticated
            await this._autoSaveFromURL(urlData);
            
            // If auto-save redirected, we won't reach here
            // Otherwise, just load the data into editor
            if (urlData.jsonString && this._aceEditor) {
                this._aceEditor.session.setValue(urlData.jsonString);
                this._updatePermalinkAnchor();
                console.log('Successfully loaded URL data into editor');
                return;
            }
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
