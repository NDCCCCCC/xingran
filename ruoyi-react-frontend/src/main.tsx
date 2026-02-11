import { createRoot } from 'react-dom/client';
import './index.css';
import './design-system/themes/theme-styles.css';
import App from './App.tsx';

createRoot(document.getElementById('root')!).render(
	<App />
);
