import { createClient } from 'https://cdn.jsdelivr.net/npm/@supabase/supabase-js/+esm';

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
    }

    connectedCallback() {
        if (this._root) return;
        this._buildRoot();
        this._initSupabase();
        this._checkAuth();
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
            this._supabase.auth.onAuthStateChange((event, session) => {
                console.log('Auth state changed:', event, session);
                if (session?.user) {
                    this._user = session.user;
                    this._showApp();
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
                this._showApp();
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

    _showApp() {
        this._loginContainer.style.display = 'none';
        this._appContainer.style.display = 'flex';
        this._appContainer.innerHTML = '';
        
        this._createHeader();
        this._createToolbar();
        this._createEditor();
        this._loadInitialData();
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

        const title = document.createElement('h2');
        title.textContent = 'Tens City';
        this._applyStyles(title, {
            margin: '0',
            fontSize: '24px',
            fontWeight: 'bold',
            color: '#24292e'
        });
        header.appendChild(title);

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

        const loadBtn = makeButton('📋 Load Objects', 'Load objects from database', () => this._loadObjects());
        const postBtn = makeButton('📤 Post Object', 'Post current JSON as new object', () => this._postObject());
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

        toolbar.appendChild(loadBtn);
        toolbar.appendChild(postBtn);
        toolbar.appendChild(clearBtn);
        toolbar.appendChild(this._permalinkAnchor);

        this._appContainer.appendChild(toolbar);
    }

    _createEditor() {
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
        this._initAceEditor(editorDiv);
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

    async _loadObjects() {
        if (!this._aceEditor) return;

        try {
            const { data, error } = await this._supabase
                .from('objects')
                .select('cid, raw, created_at, owner_uuid')
                .order('created_at', { ascending: false })
                .limit(10);

            if (error) {
                console.error('Load error:', error);
                alert('Failed to load objects: ' + error.message);
                return;
            }

            const result = {
                count: data?.length || 0,
                objects: data || []
            };

            this._aceEditor.session.setValue(JSON.stringify(result, null, 2));
        } catch (err) {
            console.error('Load exception:', err);
            alert('Failed to load objects: ' + err.message);
        }
    }

    async _postObject() {
        if (!this._aceEditor || !this._user) return;

        try {
            const text = this._aceEditor.session.getValue();
            let jsonData;
            
            try {
                jsonData = JSON.parse(text);
            } catch (parseErr) {
                alert('Invalid JSON: ' + parseErr.message);
                return;
            }

            // Validate basic JSON-LD structure
            if (!jsonData['@context']) {
                alert('JSON-LD must have @context field');
                return;
            }

            // Compute CID (simplified version - in production use proper canonicalization)
            const canonical = this._canonicalizeJSON(jsonData);
            const hash = await this._sha256(canonical);
            const cid = 'z' + this._encodeBase58(this._createCIDv1Bytes(0x0129, hash));

            // Post to database
            const { data, error } = await this._supabase
                .from('objects')
                .insert({
                    cid: cid,
                    owner_uuid: this._user.id,
                    raw: jsonData,
                    canonical: canonical
                })
                .select();

            if (error) {
                console.error('Post error:', error);
                alert('Failed to post object: ' + error.message);
                return;
            }

            alert('Object posted successfully! CID: ' + cid);
            console.log('Posted object:', data);
        } catch (err) {
            console.error('Post exception:', err);
            alert('Failed to post object: ' + err.message);
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
                const decodedData = decodeURIComponent(encodedData);
                const data = JSON.parse(decodedData);
                // Update script tag with loaded data
                const scriptTag = document.createElement('script');
                scriptTag.type = 'application/ld+json';
                scriptTag.textContent = JSON.stringify(data, null, 2);
                this.appendChild(scriptTag);
                return JSON.stringify(data, null, 2);
            } catch (err) {
                console.error('Failed to parse URL data:', err);
                return null;
            }
        }
        return null;
    }

    _updatePermalinkAnchor() {
        if (!this._aceEditor || !this._permalinkAnchor) return;

        try {
            const editorContent = this._aceEditor.session.getValue();
            // Validate JSON
            JSON.parse(editorContent);

            // Create URL with data parameter
            const url = new URL(window.location.href);
            url.searchParams.set('data', encodeURIComponent(editorContent));
            
            // Update anchor href
            this._permalinkAnchor.href = url.toString();
        } catch (err) {
            // If JSON is invalid, set href to # to prevent navigation
            this._permalinkAnchor.href = '#';
        }
    }

    async _loadInitialData() {
        // Check for permalink data in URL
        const urlData = this._loadFromURL();
        if (urlData) {
            if (this._aceEditor) {
                this._aceEditor.session.setValue(urlData);
            }
            this._updatePermalinkAnchor();
            return;
        }

        // Check for script tag data
        const scriptData = this._loadFromScriptTag();
        if (scriptData) {
            if (this._aceEditor) {
                this._aceEditor.session.setValue(scriptData);
            }
            this._updatePermalinkAnchor();
            return;
        }

        // Load some initial data from database
        await this._loadObjects();
        this._updatePermalinkAnchor();
    }

    // CID computation helpers (simplified from petri-view.js)
    _base58Alphabet = '123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz';

    _encodeBase58(bytes) {
        const alphabet = this._base58Alphabet;
        let num = 0n;
        
        for (let i = 0; i < bytes.length; i++) {
            num = num * 256n + BigInt(bytes[i]);
        }
        
        let encoded = '';
        while (num > 0n) {
            const remainder = num % 58n;
            num = num / 58n;
            encoded = alphabet[Number(remainder)] + encoded;
        }
        
        for (let i = 0; i < bytes.length && bytes[i] === 0; i++) {
            encoded = '1' + encoded;
        }
        
        return encoded;
    }

    async _sha256(data) {
        const encoder = new TextEncoder();
        const bytes = typeof data === 'string' ? encoder.encode(data) : data;
        const hashBuffer = await crypto.subtle.digest('SHA-256', bytes);
        return new Uint8Array(hashBuffer);
    }

    _createCIDv1Bytes(codec, hash) {
        const version = 0x01;
        const codecBytes = codec === 0x0129 ? [0x01, 0x29] : [codec];
        const hashType = 0x12;
        const hashLength = hash.length;
        
        const cidBytes = new Uint8Array(1 + codecBytes.length + 2 + hash.length);
        let offset = 0;
        
        cidBytes[offset++] = version;
        for (const b of codecBytes) {
            cidBytes[offset++] = b;
        }
        cidBytes[offset++] = hashType;
        cidBytes[offset++] = hashLength;
        for (let i = 0; i < hash.length; i++) {
            cidBytes[offset++] = hash[i];
        }
        
        return cidBytes;
    }

    _canonicalizeJSON(doc) {
        const canonicalize = (obj) => {
            if (obj === null || typeof obj !== 'object') {
                return JSON.stringify(obj);
            }
            
            if (Array.isArray(obj)) {
                return '[' + obj.map(item => canonicalize(item)).join(',') + ']';
            }
            
            const keys = Object.keys(obj).sort();
            const pairs = keys.map(key => {
                return JSON.stringify(key) + ':' + canonicalize(obj[key]);
            });
            return '{' + pairs.join(',') + '}';
        };
        
        return canonicalize(doc);
    }

    _applyStyles(el, styles = {}) {
        Object.assign(el.style, styles);
    }
}

customElements.define('tens-city', TensCity);
export { TensCity };
