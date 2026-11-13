import { useContext } from 'react';
import { PPPMenuNavigation } from './ppp-menu-navigation';
import { PPP_SCREENS } from './ppp-columns';
export function PPPMenuLinks({ navigate }: { navigate: () => void }) {
  const menu = useContext(PPPMenuNavigation);
  return (
    <div className="subm">
      {PPP_SCREENS.map(([screen, label]) => (
        <a
          key={screen}
          href={'#ppp/' + screen}
          className={menu?.screen === screen ? 'on' : ''}
          aria-current={menu?.screen === screen ? 'page' : undefined}
          onClick={(e) => {
            e.preventDefault();
            menu?.setScreen(screen);
            navigate();
          }}
        >
          {label}
        </a>
      ))}
    </div>
  );
}
