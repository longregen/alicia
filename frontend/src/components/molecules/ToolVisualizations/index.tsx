/**
 * Tool Visualizations
 *
 * Beautiful, specialized visualization components for each native tool type.
 * These components render tool results in a visually appealing and informative way.
 */

export { default as WebReadVisualization } from './WebReadVisualization';
export { default as WebSearchVisualization } from './WebSearchVisualization';
export { default as WebFetchVisualization } from './WebFetchVisualization';
export { default as WebLinksVisualization } from './WebLinksVisualization';
export { default as WebMetadataVisualization } from './WebMetadataVisualization';
export { default as WebScreenshotVisualization } from './WebScreenshotVisualization';
export { default as GardenTableVisualization } from './GardenTableVisualization';
export { default as GardenSQLVisualization } from './GardenSQLVisualization';
export { default as GardenSchemaVisualization } from './GardenSchemaVisualization';
export { default as ToolVisualizationRouter } from './ToolVisualizationRouter';

// Tool icons mapping
export const toolIcons: Record<string, string> = {
  web_read: '📖',
  web_fetch_raw: '🌐',
  web_fetch_structured: '🔍',
  web_search: '🔎',
  web_extract_links: '🔗',
  web_extract_metadata: '📋',
  web_screenshot: '📸',
  garden_describe_table: '📊',
  garden_execute_sql: '⚡',
  garden_schema_explore: '🗺️',
};

// Tool display names
export const toolDisplayNames: Record<string, string> = {
  web_read: 'Read Web Page',
  web_fetch_raw: 'Fetch Raw',
  web_fetch_structured: 'Fetch Structured',
  web_search: 'Web Search',
  web_extract_links: 'Extract Links',
  web_extract_metadata: 'Extract Metadata',
  web_screenshot: 'Screenshot',
  garden_describe_table: 'Describe Table',
  garden_execute_sql: 'Execute SQL',
  garden_schema_explore: 'Explore Schema',
};
