import * as yaml from 'yaml';
import type { KustomizeNode, KustomizationYaml } from '../types/kustomize.types';

interface GitUrlComponents {
    provider: 'github' | 'gitlab';
    host: string;  // Pour supporter gitlab.com et instances internes
    owner: string;
    repo: string;
    ref?: string;
    path: string;
}

export class GitCrawler {
    private nodeCounter = 0;
    private githubToken?: string;
    private gitlabToken?: string;
    private visited = new Map<string, KustomizeNode>();

    /**
     * Définir le token GitHub
     */
    setGitHubToken(token: string): void {
        this.githubToken = token;
    }

    /**
     * Définir le token GitLab
     */
    setGitLabToken(token: string): void {
        this.gitlabToken = token;
    }

    /**
     * Point d'entrée principal : crawl récursif depuis une URL d'overlay
     */
    async crawlFromOverlay(overlayUrl: string): Promise<KustomizeNode[]> {
        console.log(`\n🚀 Démarrage du crawl depuis: ${overlayUrl}`);

        this.nodeCounter = 0;
        this.visited.clear();  // ← Pas de changement ici, .clear() marche pour Map et Set

        const nodes: KustomizeNode[] = [];

        try {
            await this.crawlKustomization(overlayUrl, nodes, null);
            console.log(`\n✅ Crawl terminé: ${nodes.length} nœud(s) découvert(s)`);
            return nodes;
        } catch (error) {
            console.error('❌ Erreur lors du crawl:', error);
            throw error;
        }
    }

    private async crawlKustomization(
        url: string,
        nodes: KustomizeNode[],
        referenceType: 'resource' | 'component' | null
    ): Promise<KustomizeNode> {
        const normalizedUrl = this.normalizeUrl(url);

        // Vérifier si déjà visité - maintenant on a directement le nœud
        if (this.visited.has(normalizedUrl)) {
            const existingNode = this.visited.get(normalizedUrl)!;
            console.log(`  ⏭️ Déjà visité: ${normalizedUrl}`);
            return existingNode;
        }

        console.log(`\n🔍 Crawl: ${normalizedUrl}`);

        let kustomization: KustomizationYaml;
        let kustomizationUrl: string;
        let node: KustomizeNode;

        try {
            kustomizationUrl = this.ensureKustomizationYaml(normalizedUrl);
            const content = await this.fetchFileContent(kustomizationUrl);
            kustomization = yaml.parse(content) as KustomizationYaml;
            console.log(`  ✓ kustomization.yaml chargé`);

            node = this.createNode(normalizedUrl, kustomization, referenceType);
        } catch (error) {
            console.error(`  ⚠️ Impossible de charger kustomization.yaml:`, error);

            node = this.createErrorNode(
                normalizedUrl,
                referenceType || 'resource',
                error instanceof Error ? error.message : String(error)
            );
        }

        // Ajouter le nœud à la liste ET au cache
        nodes.push(node);
        this.visited.set(normalizedUrl, node);  // ← Stocker directement
        console.log(`  ✓ Nœud créé: ${node.id} (type: ${node.type})`);

        // Traiter resources/bases/components seulement si pas en erreur
        if (!node.error) {
            const kustomization = node.kustomizationContent;

            if (kustomization.resources && kustomization.resources.length > 0) {
                console.log(`  📦 Traitement de ${kustomization.resources.length} resource(s)...`);
                for (const resource of kustomization.resources) {
                    await this.processResource(resource, normalizedUrl, nodes);
                }
            }

            if (kustomization.bases && kustomization.bases.length > 0) {
                console.log(`  📦 Traitement de ${kustomization.bases.length} base(s) [déprécié]...`);
                for (const base of kustomization.bases) {
                    await this.processResource(base, normalizedUrl, nodes);
                }
            }

            if (kustomization.components && kustomization.components.length > 0) {
                console.log(`  🧩 Traitement de ${kustomization.components.length} component(s)...`);
                for (const component of kustomization.components) {
                    await this.processComponent(component, normalizedUrl, nodes);
                }
            }
        }

        return node;
    }



    /**
     * Traiter une resource
     */
    private async processResource(
            resource: string,
            parentUrl: string,
            nodes: KustomizeNode[]
            ): Promise<void> {
        console.log(`    📄 Resource: ${resource}`);

        // SI c'est un fichier YAML simple, IGNORER
        if (this.isYamlFile(resource)) {
            console.log(`      ⏭️ Ignoré (fichier YAML simple)`);
            return;
        }

        // Résoudre l'URL complète
        const resolvedUrl = this.resolveUrl(parentUrl, resource);
        console.log(`      → Résolu: ${resolvedUrl}`);

        // Vérifier si un kustomization.yaml existe
        try {
            await this.crawlKustomization(resolvedUrl, nodes, 'resource');
        } catch (error) {
            // Pour les resources, c'est normal qu'il n'y ait pas de kustomization
            console.warn(`      ⚠️ Pas de kustomization.yaml trouvé (ignoré)`);
            // Pas de nœud d'erreur pour les resources - c'est optionnel
        }
    }

    /**
     * Créer un nœud d'erreur
     */
    /**
     * Créer un nœud d'erreur
     */
    private createErrorNode(
        url: string,
        type: 'resource' | 'component',
        errorMessage: string
    ): KustomizeNode {
        try {
            const components = this.parseGitUrl(url);
            const displayPath = components.path || '.';

            return {
                id: `node-${this.nodeCounter++}-error`,
                path: displayPath,
                type,
                kustomizationContent: {
                    apiVersion: 'kustomize.config.k8s.io/v1beta1',
                    kind: 'Kustomization'
                } as KustomizationYaml,
                isRemote: true,
                remoteUrl: url,
                loaded: false,
                error: errorMessage
            };
        } catch (parseError) {
            // Si même le parsing échoue, créer un nœud minimal
            return {
                id: `node-${this.nodeCounter++}-error`,
                path: url,
                type,
                kustomizationContent: {
                    apiVersion: 'kustomize.config.k8s.io/v1beta1',
                    kind: 'Kustomization'
                } as KustomizationYaml,
                isRemote: true,
                remoteUrl: url,
                loaded: false,
                error: `Parse error: ${parseError}. Original: ${errorMessage}`
            };
        }
    }


    /**
     * Traiter un component
     */
    private async processComponent(
            component: string,
            parentUrl: string,
            nodes: KustomizeNode[]
            ): Promise<void> {
        console.log(`    🧩 Component: ${component}`);

        // Résoudre l'URL complète
        const resolvedUrl = this.resolveUrl(parentUrl, component);
        console.log(`      → Résolu: ${resolvedUrl}`);

        // Les components DOIVENT avoir un kustomization.yaml
        // crawlKustomization créera un nœud d'erreur si nécessaire
        await this.crawlKustomization(resolvedUrl, nodes, 'component');
    }


    /**
     * Créer un nœud
     */
    private createNode(
            url: string,
            kustomization: KustomizationYaml,
            referenceType: 'resource' | 'component' | null
            ): KustomizeNode {
        const components = this.parseGitUrl(url);
        const displayPath = components.path || '.';

        // Le type est déterminé par comment il est référencé
        // Si c'est le nœud racine (referenceType = null), on considère comme resource
        const type = referenceType || 'resource';

        return {
id: `node-${this.nodeCounter++}`,
    path: displayPath,
    type,
    kustomizationContent: kustomization,
    isRemote: true,
    remoteUrl: url,
    loaded: true
        };
    }

    /**
     * Vérifier si c'est un fichier YAML simple
     */
    private isYamlFile(path: string): boolean {
        const lower = path.toLowerCase();
        return (lower.endsWith('.yaml') || lower.endsWith('.yml')) &&
            !lower.endsWith('kustomization.yaml') &&
            !lower.endsWith('kustomization.yml');
    }

    /**
     * Vérifier si c'est une URL complète
     */
    private isFullUrl(path: string): boolean {
        return path.startsWith('http://') || path.startsWith('https://');
    }

    /**
     * Résoudre une URL (équivalent os.path.join)
     */
    private resolveUrl(baseUrl: string, relativePath: string): string {
        // SI c'est déjà une URL complète, la retourner telle quelle
        if (this.isFullUrl(relativePath)) {
            return relativePath;
        }

        // Parser l'URL de base
        const components = this.parseGitUrl(baseUrl);

        // Résoudre le chemin (comme os.path.join avec support de ..)
        const resolvedPath = this.joinPaths(components.path, relativePath);

        // Reconstruire l'URL
        return this.buildGitUrl(
                components.provider,
                components.host,
                components.owner,
                components.repo,
                components.ref,
                resolvedPath
                );
    }

    /**
     * Joindre des chemins (équivalent os.path.join avec support de ..)
     */
    private joinPaths(basePath: string, relativePath: string): string {
        // Normaliser
        const baseParts = basePath.split('/').filter(p => p && p !== '.');
        const relativeParts = relativePath.split('/').filter(p => p && p !== '.');

        // Appliquer les ".." pour remonter
        for (const part of relativeParts) {
            if (part === '..') {
                baseParts.pop();
            } else {
                baseParts.push(part);
            }
        }

        return baseParts.join('/') || '.';
    }

    /**
     * Normaliser une URL (préserver la branche pour GitLab)
     */
    private normalizeUrl(url: string): string {
        console.log('🔍 [normalizeUrl] URL avant:', url);

        const [baseUrl, queryString] = url.split('?');

        // GitLab: Extraire la branche de /-/tree/branch/path et la mettre en query param
        const gitlabTreeMatch = baseUrl.match(
                /(https?:\/\/[^\/]*gitlab[^\/]*\/[^\/]+\/[^\/]+)\/-\/(tree|blob)\/(.+)/
                );

        if (gitlabTreeMatch) {
            const repoBase = gitlabTreeMatch[1];
            const branchAndPath = gitlabTreeMatch[3];

            console.log('🔍 [normalizeUrl] GitLab tree/blob détecté');
            console.log('  Repo base:', repoBase);
            console.log('  Branch+Path:', branchAndPath);

            // Stratégie: Assumer que les 2 premiers segments = branche
            const parts = branchAndPath.split('/');
            const branch = parts.slice(0, 2).join('/');  // Ex: components/new-base
            const path = parts.slice(2).join('/');        // Ex: environments/cifmw-demo/...

            console.log('  → Branche extraite:', branch);
            console.log('  → Path extrait:', path);

            // Reconstruire l'URL avec ?ref=branch
            const pathPart = path ? `/${path}` : '';
            const normalized = `${repoBase}${pathPart}?ref=${encodeURIComponent(branch)}`;

            console.log('🔍 [normalizeUrl] URL après:', normalized);
            return normalized;
        }

        // GitHub: retirer /tree/branch
        let normalized = baseUrl.replace(/\/tree\/[^\/]+/, '');

        // Retirer trailing slash
        normalized = normalized.replace(/\/$/, '');

        // Remettre query params
        if (queryString) {
            normalized = `${normalized}?${queryString}`;
        }

        console.log('🔍 [normalizeUrl] URL après:', normalized);
        return normalized;
    }


    /**
     * S'assurer que l'URL pointe vers kustomization.yaml
     */
    private ensureKustomizationYaml(url: string): string {
        if (url.endsWith('kustomization.yaml') || url.endsWith('kustomization.yml')) {
            return url;
        }

        // Séparer l'URL des query params
        const [baseUrl, queryString] = url.split('?');

        // Ajouter /kustomization.yaml à l'URL de base
        const urlWithFile = `${baseUrl}/kustomization.yaml`;

        // Remettre les query params si présents
        return queryString ? `${urlWithFile}?${queryString}` : urlWithFile;
    }

    /**
     * Parser une URL Git (GitHub ou GitLab)
     */
    private parseGitUrl(url: string): GitUrlComponents {
        // Séparer l'URL de base et les paramètres de requête
        const [baseUrl, queryString] = url.split('?');

        // Extraire ?ref=VALUE (branch, tag ou hash)
        // IMPORTANT: Ignorer ?ref_type=heads (c'est juste informatif GitLab)
        let ref: string | undefined;
        if (queryString) {
            const refMatch = queryString.match(/ref=([^&]+)/);
            if (refMatch) {
                const refValue = decodeURIComponent(refMatch[1]);
                // Ignorer si c'est "heads" (indicateur GitLab sans valeur)
                if (refValue !== 'heads') {
                    ref = refValue;
                }
            }
        }

        // GitHub: https://github.com/owner/repo/path ou https://github.com/owner/repo?ref=branch
        const githubMatch = baseUrl.match(/https?:\/\/(github\.com)\/([^\/]+)\/([^\/]+)(?:\/(.*))?/);

        if (githubMatch) {
            return {
provider: 'github',
          host: githubMatch[1],
          owner: githubMatch[2],
          repo: githubMatch[3].replace(/\.git$/, ''),
          ref,
          path: githubMatch[4] || ''
            };
        }

        // GitLab avec /-/tree/ ou /-/blob/
        console.log('🔍 [parseGitUrl] Test regex GitLab tree/blob...');
        const gitlabTreeMatch = baseUrl.match(
                /https?:\/\/([^\/]*gitlab[^\/]*)\/([^\/]+)\/([^\/]+)\/-\/(tree|blob)\/(.+)/
                );

        if (gitlabTreeMatch) {
            const host = gitlabTreeMatch[1];
            const owner = gitlabTreeMatch[2];
            const repo = gitlabTreeMatch[3].replace(/\.git$/, '');
            const branchAndPath = gitlabTreeMatch[5];

            console.log('✅ [parseGitUrl] GitLab tree/blob URL détectée');
            console.log('  host:', host);
            console.log('  owner:', owner);
            console.log('  repo:', repo);
            console.log('  branchAndPath:', branchAndPath);

            // Si un ?ref= est fourni (et ce n'est pas "heads"), l'utiliser
            if (ref) {
                console.log('  ✅ Utilisation du ?ref= fourni:', ref);
                return {
provider: 'gitlab',
          host,
          owner,
          repo,
          ref,
          path: branchAndPath
                };
            }

            // Sinon, séparer la branche du path via l'API GitLab
            // PROBLÈME: La branche peut contenir des slashes (ex: components/new-base)
            // On doit tester progressivement du plus long au plus court
            console.log('  🔍 Détection de la branche nécessaire (testing via heuristique)');

            // Heuristique simple: tester les 2-3 premiers segments comme branche
            const parts = branchAndPath.split('/');

            // Essayer 3 segments max pour la branche (ex: feat/sub/branch)
            for (let i = Math.min(3, parts.length); i > 0; i--) {
                const potentialBranch = parts.slice(0, i).join('/');
                const potentialPath = parts.slice(i).join('/');

                console.log(`  🧪 Test: branch="${potentialBranch}", path="${potentialPath}"`);

                // Pour l'instant, on prend la première combinaison qui "semble" raisonnable
                // Idéalement, on devrait valider via l'API, mais pour la performance...

                // Si on a au moins 2 segments, c'est probablement la branche
                if (i >= 2) {
                    console.log(`  ✅ Assumé comme branche: ${potentialBranch}`);
                    return {
provider: 'gitlab',
          host,
          owner,
          repo,
          ref: potentialBranch,
          path: potentialPath
                    };
                }
            }

            // Fallback: tout est la branche, path vide
            console.warn('  ⚠️ Impossible de séparer branche/path, utilisation de tout comme branche');
            return {
provider: 'gitlab',
          host,
          owner,
          repo,
          ref: branchAndPath,
          path: ''
            };
        }

        // GitLab simplifié: https://gitlab.host/owner/repo/path
        console.log('🔍 [parseGitUrl] Test regex GitLab simple...');
        const gitlabMatch = baseUrl.match(/https?:\/\/([^\/]*gitlab[^\/]*)\/([^\/]+)\/([^\/]+)(?:\/(.*))?/);

        if (gitlabMatch) {
            console.log('✅ [parseGitUrl] GitLab URL simple détectée');
            return {
provider: 'gitlab',
          host: gitlabMatch[1],
          owner: gitlabMatch[2],
          repo: gitlabMatch[3].replace(/\.git$/, ''),
          ref: ref || 'main',  // Défaut à main si pas de ref
          path: gitlabMatch[4] || ''
            };
        }

        console.error('❌ [parseGitUrl] Aucune regex ne matche!');
        throw new Error(`URL non reconnue: ${url}. Formats supportés: GitHub et GitLab`);
    }

    /**
     * Construire une URL Git
     */
    private buildGitUrl(
            _provider: 'github' | 'gitlab',
            host: string,
            owner: string,
            repo: string,
            ref: string | undefined,
            path: string
            ): string {
        const pathPart = path ? `/${path}` : '';
        const baseUrl = `https://${host}/${owner}/${repo}${pathPart}`;

        return ref ? `${baseUrl}?ref=${encodeURIComponent(ref)}` : baseUrl;
    }

    /**
     * Télécharger le contenu d'un fichier
     */
    private async fetchFileContent(url: string): Promise<string> {
        const components = this.parseGitUrl(url);

        if (components.provider === 'github') {
            return this.fetchGitHubFile(components);
        } else if (components.provider === 'gitlab') {
            return this.fetchGitLabFile(components);
        }

        throw new Error(`Provider non supporté: ${components.provider}`);
    }

    /**
     * Télécharger depuis GitHub
     */
    private async fetchGitHubFile(components: GitUrlComponents): Promise<string> {
        const ref = components.ref || 'main';
        const apiUrl = `https://api.${components.host}/repos/${components.owner}/${components.repo}/contents/${components.path}?ref=${ref}`;

        const headers: Record<string, string> = {
            'Accept': 'application/vnd.github.v3+json'
        };

        if (this.githubToken) {
            headers['Authorization'] = `Bearer ${this.githubToken}`;
        }

        const response = await fetch(apiUrl, { headers });

        if (!response.ok) {
            if (response.status === 403) {
                const rateLimitReset = response.headers.get('X-RateLimit-Reset');
                const resetDate = rateLimitReset
                    ? new Date(parseInt(rateLimitReset) * 1000).toLocaleTimeString()
                    : 'inconnu';
                throw new Error(
                        `Rate limit GitHub atteint. Réinitialisation à ${resetDate}. ` +
                        `Ajoutez un token GitHub pour augmenter la limite à 5000 req/h.`
                        );
            }
            throw new Error(`GitHub API error: ${response.status} ${response.statusText}`);
        }

        const data = await response.json();

        // Décoder le contenu base64
        const base64Content = data.content.replace(/\n/g, '');
        const binaryString = atob(base64Content);
        const bytes = new Uint8Array(binaryString.length);
        for (let i = 0; i < binaryString.length; i++) {
            bytes[i] = binaryString.charCodeAt(i);
        }

        return new TextDecoder('utf-8').decode(bytes);
    }

    /**
     * Télécharger depuis GitLab
     */
    private async fetchGitLabFile(components: GitUrlComponents): Promise<string> {
        // Si pas de ref fourni, on doit deviner la branche
        let ref = components.ref;
        let filePath = components.path;

        // Si le path contient potentiellement une branche (ex: "components/new-base/path/to/file")
        // et qu'on n'a pas de ref explicite, on doit tester
        if (!ref && filePath.includes('/')) {
            console.log('🔍 [fetchGitLabFile] Tentative de détection de branche...');
            const result = await this.detectGitLabBranch(components.host, components.owner, components.repo, filePath);
            ref = result.branch;
            filePath = result.path;
        }

        ref = ref || 'main';

        const projectPath = encodeURIComponent(`${components.owner}/${components.repo}`);
        const encodedFilePath = encodeURIComponent(filePath);
        const apiUrl = `https://${components.host}/api/v4/projects/${projectPath}/repository/files/${encodedFilePath}/raw?ref=${encodeURIComponent(ref)}`;

        console.log(`📡 [fetchGitLabFile] API URL:`, apiUrl);
        console.log(`   Branch: ${ref}`);
        console.log(`   File: ${filePath}`);

        const headers: Record<string, string> = {};

        if (this.gitlabToken) {
            headers['PRIVATE-TOKEN'] = this.gitlabToken;
        }

        const response = await fetch(apiUrl, { headers });

        if (!response.ok) {
            throw new Error(`GitLab API error: ${response.status} - Branch: ${ref}, File: ${filePath}`);
        }

        return await response.text();
    }

    /**
     * Détecter la branche GitLab en testant progressivement
     */
    private async detectGitLabBranch(
            host: string,
            owner: string,
            repo: string,
            pathWithBranch: string
            ): Promise<{ branch: string; path: string }> {
        const parts = pathWithBranch.split('/');
        const projectPath = encodeURIComponent(`${owner}/${repo}`);

        console.log('🧪 [detectGitLabBranch] Test de', parts.length, 'combinaisons possibles');

        // Tester du plus long (3 segments max) au plus court
        for (let i = Math.min(3, parts.length); i > 0; i--) {
            const potentialBranch = parts.slice(0, i).join('/');
            const potentialPath = parts.slice(i).join('/');

            console.log(`  🧪 Test: branch="${potentialBranch}", path="${potentialPath}"`);

            try {
                // Vérifier si la branche existe via l'API
                const branchUrl = `https://${host}/api/v4/projects/${projectPath}/repository/branches/${encodeURIComponent(potentialBranch)}`;

                const headers: Record<string, string> = {};
                if (this.gitlabToken) {
                    headers['PRIVATE-TOKEN'] = this.gitlabToken;
                }

                const response = await fetch(branchUrl, { headers });

                if (response.ok) {
                    console.log(`  ✅ Branche trouvée: ${potentialBranch}`);
                    return {
branch: potentialBranch,
        path: potentialPath
                    };
                }
            } catch (error) {
                console.log(`  ❌ Erreur test branche "${potentialBranch}":`, error);
            }
        }

        // Fallback: première partie = branche, reste = path
        console.warn('  ⚠️ Aucune branche validée, utilisation heuristique');
        return {
branch: parts[0],
        path: parts.slice(1).join('/')
        };
    }



    /**
     * Scan local (conservé pour compatibilité)
     */
    async scanLocalDirectory(): Promise<KustomizeNode[]> {
        console.log('\n📁 Scan du répertoire local...');

        if (typeof window === 'undefined' || !window.electron) {
            throw new Error('Le scan local nécessite Electron');
        }

        try {
            const directoryPath = await window.electron.chooseDirectory();

            if (!directoryPath) {
                throw new Error('Aucun répertoire sélectionné');
            }

            console.log(`  📂 Répertoire: ${directoryPath}`);

            const files = await window.electron.scanDirectory(directoryPath);
            console.log(`  ✓ ${files.length} fichier(s) trouvé(s)`);

            const nodes: KustomizeNode[] = [];

            for (const filePath of files) {
                console.log(`\n  📄 Traitement: ${filePath}`);

                const content = await window.electron.readFile(filePath);

                try {
                    const kustomization = yaml.parse(content) as KustomizationYaml;

                    const relativePath = filePath
                        .replace(directoryPath, '')
                        .replace(/^[\/\\]/, '')
                        .replace(/[\/\\]kustomization\.yaml$/, '')
                        .replace(/\\/g, '/')
                            || '.';

                    const node: KustomizeNode = {
id: `node-${this.nodeCounter++}`,
    path: relativePath,
    type: 'resource',
    kustomizationContent: kustomization,
    isRemote: false,
    loaded: true
                    };

                    nodes.push(node);

                    console.log(`    ✓ Nœud créé: ${node.path} (type: ${node.type})`);
                } catch (err) {
                    console.warn(`    ⚠️ Erreur parsing YAML: ${err}`);
                }
            }

            console.log(`\n✅ Scan local terminé: ${nodes.length} nœud(s)`);
            return nodes;

        } catch (error) {
            console.error('❌ Erreur lors du scan local:', error);
            throw error;
        }
    }
}
