import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { LanguageSwitcher } from '../LanguageSwitcher/LanguageSwitcher';
import './CollapsibleSidebar.css';

interface CollapsibleSidebarProps {
  onLoadRepo: (source: string, isLocal: boolean, githubToken?: string, gitlabToken?: string) => Promise<void>;
}

export const CollapsibleSidebar: React.FC<CollapsibleSidebarProps> = ({ onLoadRepo }) => {
  const { t } = useTranslation();
  const [isCollapsed, setIsCollapsed] = useState(false);
  const [repoUrl, setRepoUrl] = useState('');
  const [githubToken, setGithubToken] = useState('');
  const [gitlabToken, setGitlabToken] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [lastLoaded, setLastLoaded] = useState<string>('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!repoUrl.trim()) return;

    setLoading(true);
    setError(null);

    try {
      await onLoadRepo(repoUrl, false, githubToken || undefined, gitlabToken || undefined);
      setLastLoaded(repoUrl);
    } catch (err) {
      const message = err instanceof Error ? err.message : t('errors.unknownError');
      setError(message);
    } finally {
      setLoading(false);
    }
  };

  const handleLoadLocal = async () => {
    setLoading(true);
    setError(null);

    try {
      await onLoadRepo('', true);
      setLastLoaded(t('sidebar.selectLocal'));
    } catch (err) {
      const message = err instanceof Error ? err.message : t('errors.unknownError');
      setError(message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className={`collapsible-sidebar left ${isCollapsed ? 'collapsed' : ''}`}>
      <button
        className="collapse-toggle"
        onClick={() => setIsCollapsed(!isCollapsed)}
        title={isCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
      >
        {isCollapsed ? '▶' : '◀'}
      </button>

      {!isCollapsed && (
        <div className="sidebar-content">
          <h1>{t('sidebar.title')}</h1>

          <LanguageSwitcher />

          <form onSubmit={handleSubmit} className="load-form">
            <label htmlFor="repo-url">{t('sidebar.repoUrlLabel')}</label>
            <input
              id="repo-url"
              type="text"
              value={repoUrl}
              onChange={(e) => setRepoUrl(e.target.value)}
              placeholder={t('sidebar.repoUrlPlaceholder')}
              disabled={loading}
            />

            <details className="token-section">
              <summary>🔑 API Tokens (optionnel)</summary>

              <label htmlFor="github-token">
                GitHub Token
                <a href="https://github.com/settings/tokens" target="_blank" rel="noopener noreferrer" className="help-link">
                  ?
                </a>
              </label>
              <input
                id="github-token"
                type="password"
                value={githubToken}
                onChange={(e) => setGithubToken(e.target.value)}
                placeholder="ghp_..."
                disabled={loading}
              />

              <label htmlFor="gitlab-token">
                GitLab Token
                <a href="https://gitlab.com/-/profile/personal_access_tokens" target="_blank" rel="noopener noreferrer" className="help-link">
                  ?
                </a>
              </label>
              <input
                id="gitlab-token"
                type="password"
                value={gitlabToken}
                onChange={(e) => setGitlabToken(e.target.value)}
                placeholder="glpat-..."
                disabled={loading}
              />

              <small className="token-help">
                Les tokens augmentent les limites: GitHub 60→5000 req/h
              </small>
            </details>

            <button type="submit" disabled={loading || !repoUrl.trim()}>
              {loading ? `🔄 ${t('sidebar.loading')}` : `🌐 ${t('sidebar.loadRemote')}`}
            </button>
          </form>

          <div className="separator">
            <span>{t('sidebar.orSeparator')}</span>
          </div>

          <button
            onClick={handleLoadLocal}
            disabled={loading}
            className="load-button local-button"
          >
            {loading ? `🔄 ${t('sidebar.loading')}` : `📁 ${t('sidebar.selectLocal')}`}
          </button>

          {lastLoaded && !error && (
            <div className="success-message">
              ✓ {t('sidebar.loadedSuccess', { source: lastLoaded })}
            </div>
          )}

          {error && (
            <div className="error-message">
              <strong>{t('sidebar.errorPrefix')}</strong> {error}
            </div>
          )}

          <div className="info-section">
            <h3>{t('sidebar.supportedSources')}</h3>
            <ul>
              <li><strong>GitHub</strong></li>
              <li><strong>GitLab</strong></li>
              <li><strong>Local</strong></li>
            </ul>
          </div>

          <div className="rate-limit-info">
            <small>{t('sidebar.rateLimitInfo')}</small>
          </div>
        </div>
      )}
    </div>
  );
};
