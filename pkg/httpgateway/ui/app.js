const { createApp, reactive, ref, computed, watch, onMounted, nextTick } = Vue;

const app = createApp({
  data() {
    return {
      // Nav
      templates: [], loadingNav: true, openServices: {},
      selectedService: null, selectedGroup: null,
      // Scope
      scope: 'SYSTEM', projectId: '', storeId: '', userId: '',
      // Config
      currentTemplate: null, currentConfig: null,
      fieldValues: {}, dirty: false, loadingContent: false,
      secretVisible: {},
      jsonErrors: {},
      // Topbar
      userName: '', groupTitle: 'Select a configuration group',
      // History
      showHistory: false, historyItems: [], loadingHistory: false,
      viewingVersion: null,
      // Publish
      showPublishModal: false, publishingVersion: null, publishing: false,
      // Save
      saving: false,
      // Toast
      toasts: [], toastId: 0,
      // Template panel
      showTmplPanel: false, tmplTab: 'import', importMode: 'paste',
      uploadedFileName: '', validationOutput: '', importResults: [], importing: false,
      // Manage
      manageTemplates: [], loadingManage: false,
      // Overlay
      showOverlay: false,
    };
  },
  computed: {
    serviceMap() {
      const map = {};
      for (const t of this.templates) {
        if (!map[t.serviceName]) map[t.serviceName] = { label: t.serviceLabel || t.serviceName, groups: [] };
        map[t.serviceName].groups.push(t);
      }
      return map;
    },
    scopeBadgeText() {
      const parts = [this.scope];
      if (this.projectId) parts.push('proj:'+this.projectId);
      if (this.storeId) parts.push('store:'+this.storeId);
      if (this.userId) parts.push('user:'+this.userId);
      return parts.join(' · ');
    },
    visibleFields() {
      if (!this.currentTemplate) return [];
      return (this.currentTemplate.fields||[]).filter(f => {
        if (!f.displayOn || f.displayOn.length === 0) return true;
        return f.displayOn.includes(this.scope);
      });
    },
  },
  methods: {
    // API
    async api(method, path, body) {
      const opts = { method, headers: { 'Content-Type': 'application/json' }, cache: 'no-store' };
      if (body) opts.body = JSON.stringify(body);
      const resp = await fetch(path, opts);
      const json = await resp.json().catch(() => null);
      if (!resp.ok) throw new Error(json?.message || json?.error || `HTTP ${resp.status}`);
      return json;
    },
    toast(msg, type='info') {
      const id = ++this.toastId;
      this.toasts.push({ id, msg, type });
      setTimeout(() => { this.toasts = this.toasts.filter(t => t.id !== id); }, 4000);
    },
    sanitizeId(path) { return path.replace(/[^a-zA-Z0-9]/g, '_'); },
    formatDate(iso) {
      if (!iso) return '—';
      return new Date(iso).toLocaleString(undefined, { year:'numeric', month:'short', day:'numeric', hour:'2-digit', minute:'2-digit' });
    },
    getUsername() { return this.userName.trim() || 'admin'; },
    scopeParams() {
      const params = new URLSearchParams();
      if (['PROJECT','STORE','USER'].includes(this.scope) && this.projectId) params.set('projectId', this.projectId);
      if (['STORE','USER'].includes(this.scope) && this.storeId) params.set('storeId', this.storeId);
      if (this.scope === 'USER' && this.userId) params.set('userId', this.userId);
      return params;
    },

    // Sidebar
    toggleService(svc) { this.openServices[svc] = !this.openServices[svc]; },
    async loadTemplates() {
      this.loadingNav = true;
      try {
        const data = await this.api('GET', '/api/v1/config/templates');
        this.templates = data.templates || [];
        // Auto-open selected service
        if (this.selectedService) this.openServices[this.selectedService] = true;
      } catch(e) { this.templates = []; this.toast('Failed to load templates: '+e.message, 'error'); }
      this.loadingNav = false;
    },

    // Select group
    async selectGroup(svc, grp) {
      this.selectedService = svc;
      this.selectedGroup = grp;
      this.dirty = false;
      this.fieldValues = {};
      this.secretVisible = {};
      this.jsonErrors = {};
      this.openServices[svc] = true;
      await this.loadConfig();
    },

    onScopeChange() { if (this.selectedService && this.selectedGroup) this.loadConfig(); },
    onScopeIdChange() { if (this.selectedService && this.selectedGroup) this.loadConfig(); },

    async loadConfig() {
      if (!this.selectedService || !this.selectedGroup) return;
      this.viewingVersion = null;
      this.loadingContent = true;
      const params = this.scopeParams();
      try {
        const tParams = new URLSearchParams({ groupId: this.selectedGroup });
        const cParams = new URLSearchParams(params);
        cParams.set('groupId', this.selectedGroup);
        const [tmpl, cfg] = await Promise.all([
          this.api('GET', `/api/v1/config/${this.selectedService}/template?${tParams}`),
          this.api('GET', `/api/v1/config/${this.selectedService}/scope/${this.scope}/latest?${cParams}`).catch(() => null)
        ]);
        this.currentTemplate = tmpl;
        this.currentConfig = cfg;
        this.fieldValues = {};
        this.secretVisible = {};
        this.jsonErrors = {};
        this.dirty = false;
        this.groupTitle = tmpl.groupLabel || this.selectedGroup;
        // Initialize field values
        const cfgFields = cfg?.fields || {};
        for (const f of (tmpl.fields||[])) {
          const raw = cfgFields[f.path] !== undefined ? cfgFields[f.path] : f.defaultValue ?? '';
          this.fieldValues[f.path] = String(raw);
          if (f.type === 'SECRET') this.secretVisible[f.path] = false;
        }
      } catch(e) { this.toast('Error: '+e.message, 'error'); this.groupTitle = this.selectedGroup; }
      this.loadingContent = false;
    },

    // Field handlers
    onFieldInput(path, val) { this.fieldValues[path] = val; this.dirty = true; },
    onToggle(path, evt) { this.fieldValues[path] = evt.target.checked ? 'true' : 'false'; this.dirty = true; },
    formatJsonValue(path) {
      const v = this.fieldValues[path] || '';
      try { if (v.trim()) return JSON.stringify(JSON.parse(v), null, 2); } catch(e) {}
      return v;
    },
    onJsonInput(path, evt) {
      this.fieldValues[path] = evt.target.value;
      this.dirty = true;
      try { if (evt.target.value.trim()) JSON.parse(evt.target.value); delete this.jsonErrors[path]; }
      catch(e) { this.jsonErrors[path] = e.message; }
    },
    formatJsonField(path) {
      try {
        const el = document.getElementById('field-'+this.sanitizeId(path));
        if (el) { el.value = JSON.stringify(JSON.parse(el.value), null, 2); this.fieldValues[path] = el.value; }
      } catch(e) { alert('Invalid JSON: '+e.message); }
    },
    arrayToLines(path) {
      const v = this.fieldValues[path] || '[]';
      try { const a = JSON.parse(v); if (Array.isArray(a)) return a.join('\n'); } catch(e) {}
      return v;
    },
    onArrayInput(path, evt) {
      const arr = evt.target.value.split('\n').map(s => s.trim()).filter(s => s.length > 0);
      this.fieldValues[path] = JSON.stringify(arr);
      this.dirty = true;
    },

    // Save
    async saveConfig() {
      this.saving = true;
      const params = this.scopeParams();
      try {
        const body = { fields: {...this.fieldValues}, userName: this.getUsername() };
        if (params.get('projectId')) body.projectId = params.get('projectId');
        if (params.get('storeId')) body.storeId = params.get('storeId');
        if (params.get('userId')) body.userId = params.get('userId');
        const cParams = new URLSearchParams({ groupId: this.selectedGroup });
        const result = await this.api('PUT', `/api/v1/config/${this.selectedService}/scope/${this.scope}?${cParams}`, body);
        this.currentConfig = result;
        this.dirty = false;
        this.toast(`Saved as draft version ${result.latestVersion}`, 'success');
      } catch(e) { this.toast('Save failed: '+e.message, 'error'); }
      this.saving = false;
    },

    // Publish
    startPublish() {
      if (!this.currentConfig?.latestVersion) { this.toast('Save a draft first', 'error'); return; }
      this.publishingVersion = this.currentConfig.latestVersion;
      this.showPublishModal = true;
    },
    async confirmPublish() {
      this.publishing = true;
      const params = this.scopeParams();
      try {
        const body = { version: this.publishingVersion, userName: this.getUsername() };
        if (params.get('projectId')) body.projectId = params.get('projectId');
        if (params.get('storeId')) body.storeId = params.get('storeId');
        if (params.get('userId')) body.userId = params.get('userId');
        const cParams = new URLSearchParams({ groupId: this.selectedGroup });
        await this.api('POST', `/api/v1/config/${this.selectedService}/scope/${this.scope}/publish?${cParams}`, body);
        this.toast(`Version ${this.publishingVersion} published!`, 'success');
        this.showPublishModal = false;
        await this.loadConfig();
      } catch(e) { this.toast('Publish failed: '+e.message, 'error'); }
      this.publishing = false;
    },
    async publishVersion(version) {
      try {
        const params = this.scopeParams();
        const body = { version, userName: this.getUsername() };
        if (params.get('projectId')) body.projectId = params.get('projectId');
        const cParams = new URLSearchParams({ groupId: this.selectedGroup });
        await this.api('POST', `/api/v1/config/${this.selectedService}/scope/${this.scope}/publish?${cParams}`, body);
        this.toast(`Version ${version} published!`, 'success');
        this.closeHistoryPanel();
        this.loadConfig();
      } catch(e) { this.toast('Failed: '+e.message, 'error'); }
    },

    // History
    async openHistoryPanel() {
      this.showHistory = true;
      this.showOverlay = true;
      this.loadingHistory = true;
      const params = this.scopeParams();
      const cParams = new URLSearchParams(params);
      cParams.set('groupId', this.selectedGroup);
      cParams.set('limit', '20');
      try {
        const data = await this.api('GET', `/api/v1/config/${this.selectedService}/scope/${this.scope}/history?${cParams}`);
        this.historyItems = data.history || [];
      } catch(e) { this.historyItems = []; this.toast(e.message, 'error'); }
      this.loadingHistory = false;
    },
    closeHistoryPanel() { this.showHistory = false; this.showOverlay = false; },

    async viewVersion(version) {
      this.viewingVersion = version;
      this.closeHistoryPanel();
      this.loadingContent = true;
      const params = this.scopeParams();
      const cParams = new URLSearchParams(params);
      cParams.set('groupId', this.selectedGroup);
      try {
        const cfg = await this.api('GET', `/api/v1/config/${this.selectedService}/scope/${this.scope}/version/${version}?${cParams}`);
        const cfgFields = cfg?.fields || {};
        for (const f of (this.currentTemplate?.fields||[])) {
          const raw = cfgFields[f.path] !== undefined ? cfgFields[f.path] : f.defaultValue ?? '';
          this.fieldValues[f.path] = String(raw);
        }
      } catch(e) { this.toast('Error loading v'+version+': '+e.message, 'error'); }
      this.loadingContent = false;
    },

    // Template panel
    openTemplatePanel() {
      this.showTmplPanel = true; this.showOverlay = true;
      this.showHistory = false;
      this.$nextTick(() => this.initMonaco());
    },
    closeTmplPanel() { this.showTmplPanel = false; this.showOverlay = false; },
    switchToManage() { this.tmplTab = 'manage'; this.loadTmplManageList(); },
    closeAll() {
      this.closeHistoryPanel();
      this.closeTmplPanel();
      this.showPublishModal = false;
    },

    // Monaco
    initMonaco() {
      if (window._monacoEditor) return;
      const container = document.getElementById('editor-container');
      if (!container || !window.monaco) return;
      const defaultYaml = `# Paste your YAML template here...\n#\n# Example:\n# service:\n#   id: my-service\n#   label: My Service\n# groups:\n#   - id: my-group\n#     label: My Group\n#     fields:\n#       - path: my.setting\n#         label: My Setting\n#         type: STRING\n#         defaultValue: hello\n#         displayOn:\n#           - SYSTEM\n`;
      window._monacoEditor = monaco.editor.create(container, {
        value: defaultYaml, language: 'yaml', theme: 'vs',
        automaticLayout: true, minimap: { enabled: false },
        tabSize: 2, insertSpaces: true, fontSize: 13,
        fontFamily: "'SFMono-Regular', Consolas, monospace",
        scrollBeyondLastLine: false, lineNumbersMinChars: 3,
        padding: { top: 12, bottom: 12 }
      });
      window._monacoEditor.onDidChangeModelContent(() => {
        clearTimeout(window._yamlTimer);
        window._yamlTimer = setTimeout(() => this.validateYamlOnly(), 1000);
      });
    },
    getYamlText() { return window._monacoEditor ? window._monacoEditor.getValue().trim() : ''; },

    handleFileUpload(evt) {
      const file = evt.target.files[0];
      if (!file) return;
      this.uploadedFileName = file.name;
      const reader = new FileReader();
      reader.onload = e => {
        if (window._monacoEditor) window._monacoEditor.setValue(e.target.result);
        this.importMode = 'paste';
        this.toast('File "'+file.name+'" loaded into editor', 'info');
      };
      reader.readAsText(file);
    },

    // Validation
    validateYamlOnly() {
      const text = this.getYamlText();
      if (!text) { this.toast('Paste YAML content first', 'error'); return; }
      try {
        const parsed = jsyaml.load(text);
        if (!parsed || typeof parsed !== 'object') throw new Error('YAML must map to a top-level object');
        const { errors, warnings } = this.validateTemplate(parsed);
        this.showValidation(parsed, errors, warnings);
      } catch(e) {
        this.validationOutput = `<div class="bg-red-50 border border-red-200 rounded-lg p-3 text-[12px]"><div class="text-danger">✕ Parse error: ${e.message}</div></div>`;
      }
    },
    validateTemplate(parsed) {
      // ADDED HTML AND TEXTAREA HERE
      const VALID_TYPES = ['STRING','INT','FLOAT','BOOLEAN','JSON','ARRAY_STRING','SECRET', 'HTML', 'TEXTAREA'];
      const VALID_SCOPES = ['SYSTEM','PROJECT','STORE','USER'];
      const errors = [], warnings = [];
      if (!parsed.service) { errors.push('Missing top-level "service:" key'); return {errors,warnings}; }
      if (!parsed.service.id) errors.push('service.id is required');
      if (!parsed.service.label) warnings.push('service.label is not set');
      if (!parsed.groups || !Array.isArray(parsed.groups) || !parsed.groups.length) {
        errors.push('At least one group is required under "groups:"'); return {errors,warnings};
      }
      parsed.groups.forEach((grp,gi) => {
        const pfx = `Group[${gi}] "${grp.id||'?'}"`;
        if (!grp.id) errors.push(pfx+': id is required');
        if (!grp.label) warnings.push(pfx+': label is not set');
        if (!grp.fields || !grp.fields.length) { warnings.push(pfx+': no fields defined'); return; }
        grp.fields.forEach((f,fi) => {
          const fp = pfx+` Field[${fi}] "${f.path||'?'}"`;
          if (!f.path) errors.push(fp+': path is required');
          if (!f.type) errors.push(fp+': type is required');
          else if (!VALID_TYPES.includes(f.type.toUpperCase())) errors.push(fp+': unknown type "'+f.type+'"');
          (f.displayOn||[]).forEach(s => { if (!VALID_SCOPES.includes(s.toUpperCase())) warnings.push(fp+': unknown scope "'+s+'"'); });
        });
      });
      return {errors, warnings};
    },
    showValidation(parsed, errors, warnings) {
      const cls = errors.length ? 'bg-red-50 border-red-200' : 'bg-green-50 border-green-200';
      let html = `<div class="${cls} border rounded-lg p-3 text-[12px]">`;
      if (!errors.length && !warnings.length) {
        const gc = (parsed.groups||[]).length;
        const fc = (parsed.groups||[]).reduce((s,g) => s + (g.fields||[]).length, 0);
        html += `<div class="text-ok">✓ Valid: ${gc} group(s), ${fc} field(s)</div>`;
      }
      errors.forEach(e => { html += `<div class="text-danger">✕ ${e}</div>`; });
      warnings.forEach(w => { html += `<div class="text-warn">⚠ ${w}</div>`; });
      html += '</div>';
      this.validationOutput = html;
    },

    // Import
    async applyImport() {
      const text = this.getYamlText();
      if (!text) { this.toast('Paste or upload a YAML file first', 'error'); return; }
      let parsed;
      try { parsed = jsyaml.load(text); if (!parsed || typeof parsed !== 'object') throw new Error('Invalid'); }
      catch(e) { this.toast('YAML parse error: '+e.message, 'error'); return; }
      const {errors} = this.validateTemplate(parsed);
      if (errors.length) { this.toast('Fix '+errors.length+' error(s) before applying', 'error'); return; }
      this.importing = true;
      const body = {
        service: { id: parsed.service.id, label: parsed.service.label||parsed.service.id },
        groups: (parsed.groups||[]).map(g => ({
          id: g.id, label: g.label||g.id, description: g.description||'',
          sortOrder: parseInt(g.sortOrder)||0,
          fields: (g.fields||[]).map((f,i) => ({
            path: f.path, label: f.label||f.path, description: f.description||'',
            type: (f.type||'STRING').toUpperCase(), defaultValue: f.defaultValue||'',
            sortOrder: parseInt(f.sortOrder)||(i*100),
            displayOn: (f.displayOn||[]).map(s => s.toUpperCase()),
            options: (f.options||[]).map(o => ({value:o.value||'',label:o.label||o.value||''})),
          }))
        })),
        userName: this.getUsername(),
      };
      try {
        const res = await fetch('/api/v1/config/templates', { method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(body) });
        const data = await res.json();
        this.importResults = data.results || [];
        const ok = this.importResults.filter(r => r.status==='ok').length;
        const fail = this.importResults.filter(r => r.status!=='ok').length;
        if (!fail) { this.toast('✓ '+ok+' group(s) imported', 'success'); await this.loadTemplates(); }
        else this.toast(ok+' ok, '+fail+' failed', 'error');
      } catch(e) { this.toast('Import failed: '+e.message, 'error'); }
      this.importing = false;
    },

    // Manage
    async loadTmplManageList() {
      this.loadingManage = true;
      try {
        const res = await fetch('/api/v1/config/templates?includeInactive=true', { cache:'no-store' });
        const data = await res.json();
        this.manageTemplates = data.templates || [];
      } catch(e) { this.manageTemplates = []; this.toast(e.message, 'error'); }
      this.loadingManage = false;
    },
    async toggleTemplateActive(svc, grp, active, checkbox) {
      checkbox.disabled = true;
      try {
        const res = await fetch(`/api/v1/config/templates/${encodeURIComponent(svc)}/${encodeURIComponent(grp)}/active`,
          { method:'PATCH', headers:{'Content-Type':'application/json'}, body:JSON.stringify({active}) });
        if (!res.ok) { const err = await res.json().catch(()=>({})); throw new Error(err.message||'Request failed'); }
        this.toast('Template "'+grp+'" '+(active?'enabled':'disabled'), 'success');
        await this.loadTemplates();
        await this.loadTmplManageList();
      } catch(e) { checkbox.checked = !active; this.toast('Toggle failed: '+e.message, 'error'); }
      checkbox.disabled = false;
    },
  },
  mounted() { this.loadTemplates(); }
});

app.component('html-editor', {
  props: ['modelValue', 'readonly', 'id'],
  emits: ['update:modelValue'],
  template: `
    <div class="border border-surface-200 rounded-lg overflow-hidden flex flex-col bg-white h-[400px]">
      <div class="flex items-center gap-1 bg-surface-50 border-b border-surface-200 px-2 py-1.5 shrink-0">
        <button type="button" @click="tab='code'" class="px-3 py-1 text-[12px] rounded-md font-medium transition-colors" :class="tab==='code' ? 'bg-white shadow-sm border border-surface-200 text-brand-600' : 'text-surface-500 hover:text-surface-700'">Code</button>
        <button type="button" @click="tab='preview'" class="px-3 py-1 text-[12px] rounded-md font-medium transition-colors" :class="tab==='preview' ? 'bg-white shadow-sm border border-surface-200 text-brand-600' : 'text-surface-500 hover:text-surface-700'">Preview</button>
      </div>
      <div v-show="tab === 'code'" class="flex-1 w-full relative min-h-0">
        <div ref="editorContainer" class="absolute inset-0"></div>
      </div>
      <div v-show="tab === 'preview'" class="flex-1 overflow-auto bg-surface-50 min-h-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSI4IiBoZWlnaHQ9IjgiPgo8cmVjdCB3aWR0aD0iNCIgaGVpZ2h0PSI0IiBmaWxsPSIjZTVlN2ViIiAvPgo8cmVjdCB4PSI0IiB5PSI0IiB3aWR0aD0iNCIgaGVpZ2h0PSI0IiBmaWxsPSIjZTVlN2ViIiAvPgo8L3N2Zz4=')]">
        <div class="bg-white min-h-[100%] mx-auto shadow-sm">
          <iframe ref="previewFrame" class="w-full h-full min-h-[350px] border-none block" :srcdoc="modelValue" sandbox="allow-same-origin"></iframe>
        </div>
      </div>
    </div>
  `,
  data() {
    return { tab: 'code' };
  },
  watch: {
    modelValue(val) {
      if (this._editor && val !== this._editor.getValue()) {
        this._editor.setValue(val);
      }
    },
    readonly(val) {
      if (this._editor) this._editor.updateOptions({ readOnly: val });
    },
    tab(val) {
      if (val === 'code') {
        this.$nextTick(() => { if (this._editor) this._editor.layout(); });
      }
    }
  },
  mounted() {
    if (!window.monaco) return;
    this._editor = monaco.editor.create(this.$refs.editorContainer, {
      value: this.modelValue || '',
      language: 'html',
      theme: 'vs',
      automaticLayout: true,
      minimap: { enabled: false },
      tabSize: 2,
      fontSize: 13,
      fontFamily: "'SFMono-Regular', Consolas, monospace",
      scrollBeyondLastLine: false,
      lineNumbersMinChars: 3,
      padding: { top: 12, bottom: 12 },
      readOnly: !!this.readonly
    });
    this._editor.onDidChangeModelContent(() => {
      this.$emit('update:modelValue', this._editor.getValue());
    });
  },
  beforeUnmount() {
    if (this._editor) this._editor.dispose();
  }
});

app.mount('#app');
