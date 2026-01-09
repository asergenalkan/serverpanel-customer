import React from 'react';
import Lottie from 'lottie-react';
import animationData from '@/assets/loading-dots-blue.json';

interface LoadingAnimationProps {
  message?: string;
  heightClass?: string; // Tailwind height class for the container
}

const LoadingAnimation: React.FC<LoadingAnimationProps> = ({
  message = 'Yükleniyor...',
  heightClass = 'h-64',
}) => {
  return (
    <div className={`flex flex-col items-center justify-center ${heightClass} gap-4 text-muted-foreground`}>
      <div className="w-24 h-24">
        <Lottie animationData={animationData} loop autoplay />
      </div>
      {message && <p className="text-sm">{message}</p>}
    </div>
  );
};

export default LoadingAnimation;
