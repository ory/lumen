import React from 'react';

export function Button(): JSX.Element {
  return <button>Click</button>;
}

export const Card = ({ title }: { title: string }) => {
  return <div>{title}</div>;
};

export class App extends React.Component {
  render(): JSX.Element {
    return <div><Button /><Card title="test" /></div>;
  }
}

export interface Props {
  name: string;
}

export type Theme = "light" | "dark";

// Component defined inside another (edge case)
function Wrapper() {
  function Inner() {
    return <span>inner</span>;
  }
  return <div><Inner /></div>;
}
